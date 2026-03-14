package tasks

import "helios.io/cue/definitions/tekton"

// Git Update Manifest Task
#GitUpdateManifest: tekton.#TektonTask & {
	parameter: {
		name: "git-update-manifest"
	}

	// Alias config for internal use
	_config: tekton.#Defaults

	output: spec: {
		params: [
			tekton.#CommonParams.gitops.repoUrl,
			tekton.#CommonParams.gitops.manifestPath,
			tekton.#CommonParams.gitops.newImageUrl,
			tekton.#CommonParams.gitops.branch,
			tekton.#CommonParams.gitops.secret, {
			name:    "REPLICAS"
			default: "2"
			type:    "string"
		}, {
			name:    "PORT"
			default: "8080"
			type:    "string"
		}]
		workspaces: [{
			name: "gitops-repo"
		}]
		steps: [{
			// Step 1: Clone Repo using Git image
			name:  "git-clone"
			image: _config.images.gitClone
			script: """
				#!/bin/sh
				set -e
				
				# Clean workspace
				cd "$(workspaces.gitops-repo.path)"
				rm -rf ./*
				rm -rf ./.??*
				rm -rf .git
				
				# Clone
				echo "Cloning $(params.GITOPS_REPO_URL) to current dir..."
				git clone "$(params.GITOPS_REPO_URL)" .
				git checkout -B "$(params.GITOPS_REPO_BRANCH)"
				"""
		}, {
			// Step 2: Update Manifests using YQ image
			name:  "update-manifests"
			image: _config.images.yq
			securityContext: {
				runAsUser: 0
			}
			entrypoint: ["/bin/sh"] // Override default yq entrypoint
			script: """
				#!/bin/sh
				set -e
				cd "$(workspaces.gitops-repo.path)"

				export IMAGE_URL="$(params.NEW_IMAGE_URL)"
				export REPLICAS="$(params.REPLICAS)"
				export PORT="$(params.PORT)"
				MANIFEST_PATH="$(params.MANIFEST_PATH)"

				# Logic tạo file tự động
				if echo "$MANIFEST_PATH" | grep -qvE '\\.ya?ml$'; then
				    echo "Path '$MANIFEST_PATH' treated as DIRECTORY."
				    mkdir -p "$MANIFEST_PATH"
				    
				    DEP_FILE="$MANIFEST_PATH/deployment.yaml"
				    SVC_FILE="$MANIFEST_PATH/service.yaml"
				    MANIFEST_FILES="$DEP_FILE $SVC_FILE"
				    APP_NAME=$(basename "$MANIFEST_PATH")
				
				    if [ ! -f "$DEP_FILE" ]; then
				        echo "Creating default manifests..."
				        printf "apiVersion: apps/v1\\nkind: Deployment\\nmetadata:\\n  name: ${APP_NAME}\\nspec:\\n  replicas: ${REPLICAS}\\n  selector:\\n    matchLabels:\\n      app: ${APP_NAME}\\n  template:\\n    metadata:\\n      labels:\\n        app: ${APP_NAME}\\n    spec:\\n      containers:\\n        - name: app\\n          image: ${IMAGE_URL}\\n          ports:\\n            - containerPort: ${PORT}\\n" > "$DEP_FILE"
				
				        printf "apiVersion: v1\\nkind: Service\\nmetadata:\\n  name: ${APP_NAME}\\nspec:\\n  selector:\\n    app: ${APP_NAME}\\n  ports:\\n    - protocol: TCP\\n      port: ${PORT}\\n      targetPort: ${PORT}\\n  type: ClusterIP\\n" > "$SVC_FILE"
				    fi
				else
				    echo "Path '$MANIFEST_PATH' treated as FILE."
				    mkdir -p "$(dirname "$MANIFEST_PATH")"
				    MANIFEST_FILES="$MANIFEST_PATH"
				    APP_NAME=$(basename "$MANIFEST_PATH" | sed 's/\\.[^.]*$//')
				
				    if [ ! -f \"$MANIFEST_PATH\" ]; then
				        echo "Creating combined manifest file..."
				        printf "apiVersion: apps/v1\\nkind: Deployment\\nmetadata:\\n  name: ${APP_NAME}\\nspec:\\n  replicas: ${REPLICAS}\\n  selector:\\n    matchLabels:\\n      app: ${APP_NAME}\\n  template:\\n    metadata:\\n      labels:\\n        app: ${APP_NAME}\\n    spec:\\n      containers:\\n        - name: app\\n          image: ${IMAGE_URL}\\n          ports:\\n            - containerPort: ${PORT}\\n---\\napiVersion: v1\\nkind: Service\\nmetadata:\\n  name: ${APP_NAME}\\nspec:\\n  selector:\\n    app: ${APP_NAME}\\n  ports:\\n    - protocol: TCP\\n      port: ${PORT}\\n      targetPort: ${PORT}\\n  type: ClusterIP\\n" > "$MANIFEST_PATH"
				    fi
				fi
				
				# Update manifests using yq
				for FILE in $MANIFEST_FILES; do
				  if [ -f "$FILE" ]; then
				      echo "Updating $FILE..."
				      yq -i 'select(.kind == "Deployment") .spec.template.spec.containers[].image = env(IMAGE_URL)' "$FILE"
				      yq -i 'select(.kind == "Deployment") .spec.replicas = env(REPLICAS)' "$FILE"
				      yq -i 'select(.kind == "Deployment") .spec.template.spec.containers[].ports[0].containerPort = env(PORT)' "$FILE"
				      yq -i 'select(.kind == "Service") .spec.ports[0].targetPort = env(PORT)' "$FILE"
				  fi
				done
				"""
		}, {
			// Step 3: Commit and Push using Git image
			name:  "git-commit"
			image: _config.images.gitClone
			envFrom: [{
				secretRef: {
					name:     "$(params.GITOPS_SECRET)"
					optional: true
				}
			}]
			script: """
				#!/bin/sh
				set -e
				cd "$(workspaces.gitops-repo.path)"

				git config user.email "tekton-pipeline@helios.dev"
				git config user.name "Tekton Pipeline"

				if [ -n "${username:-}" ] && [ -n "${password:-}" ]; then
				  RAW_URL=$(echo "$(params.GITOPS_REPO_URL)" | sed 's|https://.*@|https://|')
				  REPO_URL_WITH_AUTH="$(echo "$RAW_URL" | sed "s|https://|https://${username}:${password}@|")"
				  git remote set-url origin "${REPO_URL_WITH_AUTH}"
				else
				    echo "WARNING: username or password env vars not set. Push might fail."
				fi

				git add .
				if git diff-index --quiet HEAD --; then
				    echo "No changes to commit"
				else
				    git commit -m "chore: Update image=$(params.NEW_IMAGE_URL) [skip-ci]"
				    git push origin "$(params.GITOPS_REPO_BRANCH)"
				fi
				"""
		}]
	}
}
