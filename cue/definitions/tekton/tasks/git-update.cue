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
			tekton.#CommonParams.gitops.secretRef,
			tekton.#CommonParams.gitops.authorName,
			tekton.#CommonParams.gitops.authorEmail, {
			name:    "replicas"
			default: "2"
			type:    "string"
		}, {
			name:    "port"
			default: "8080"
			type:    "string"
		}, {
			name:    "image-type"
			default: "app"
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
				if ! command -v git >/dev/null 2>&1; then
					apk add --no-cache git
				fi
				
				WORKSPACE_PATH="$(workspaces.gitops-repo.path)"
				cd "$WORKSPACE_PATH"

				# Tekton PVC ownership can differ from the running UID.
				# Force git to trust this workspace path.
				git_safe() {
					git -c safe.directory="$WORKSPACE_PATH" "$@"
				}

				# Prepare repo content using git primitives (no rm dependency).
				echo "Cloning $(params.gitops-repo-url) to current dir..."
				if [ -d .git ]; then
					git_safe remote remove origin || true
				else
					git_safe init
				fi
				git_safe remote add origin "$(params.gitops-repo-url)"
				git_safe fetch --depth=1 origin "$(params.gitops-repo-branch)"
				git_safe checkout -B "$(params.gitops-repo-branch)" FETCH_HEAD
				git_safe reset --hard FETCH_HEAD
				git_safe clean -fdx

				# Ensure subsequent steps can modify files when chmod is available.
				# Some minimal git images do not ship coreutils.
				if command -v chmod >/dev/null 2>&1; then
					chmod -R a+rwX "$WORKSPACE_PATH"
				fi
				"""
		}, {
			// Step 2: Update Manifests using YQ image
			name:  "update-manifests"
			image: _config.images.yq
			securityContext: {
				runAsUser: 0
			}
			script: """
				#!/bin/sh
				set -e
				cd "$(workspaces.gitops-repo.path)"

				export IMAGE_URL="$(params.new-image-url)"
				export REPLICAS="$(params.replicas)"
				export PORT="$(params.port)"
				export IMAGE_TYPE="$(params.image-type)"

				# Defensive defaults: avoid generating invalid manifests when inputs are empty/0
				if [ -z "${REPLICAS}" ] || [ "${REPLICAS}" = "0" ]; then
				  export REPLICAS="1"
				fi
				if [ -z "${PORT}" ] || [ "${PORT}" = "0" ]; then
				  export PORT="8080"
				fi
				MANIFEST_PATH="$(params.manifest-path)"

				# Logic tạo file tự động
				if echo "$MANIFEST_PATH" | grep -qvE '\\.ya?ml$'; then
				    echo "Path '$MANIFEST_PATH' treated as DIRECTORY."
				    mkdir -p "$MANIFEST_PATH"

				    # If the operator already renders a combined manifest.yaml in this directory,
				    # prefer updating that file rather than creating separate default manifests.
				    COMBINED_FILE="$MANIFEST_PATH/manifest.yaml"
				    if [ "$IMAGE_TYPE" = "migrate" ]; then
				        if [ -f "presync-job.yaml" ]; then
				            MANIFEST_FILES="presync-job.yaml"
				        else
				            MANIFEST_FILES="$MANIFEST_PATH/presync-job.yaml"
				        fi
				    elif [ -f "$COMBINED_FILE" ]; then
				        MANIFEST_FILES="$COMBINED_FILE"
				    else
				        DEP_FILE="$MANIFEST_PATH/deployment.yaml"
				            SVC_FILE="$MANIFEST_PATH/service.yaml"
				            MANIFEST_FILES="$DEP_FILE $SVC_FILE"
				            APP_NAME=$(basename "$MANIFEST_PATH")

				            if [ ! -f "$DEP_FILE" ]; then
				                echo "Creating default manifests..."
				                printf "apiVersion: apps/v1\\nkind: Deployment\\nmetadata:\\n  name: ${APP_NAME}\\nspec:\\n  replicas: ${REPLICAS}\\n  selector:\\n    matchLabels:\\n      app: ${APP_NAME}\\n  template:\\n    metadata:\\n      labels:\\n        app: ${APP_NAME}\\n    spec:\\n      containers:\\n        - name: app\\n          image: ${IMAGE_URL}\\n          ports:\\n            - containerPort: ${PORT}\\n" > "$DEP_FILE"

				                printf "apiVersion: v1\\nkind: Service\\nmetadata:\\n  name: ${APP_NAME}\\nspec:\\n  selector:\\n    app: ${APP_NAME}\\n  ports:\\n    - protocol: TCP\\n      port: ${PORT}\\n      targetPort: ${PORT}\\n  type: ClusterIP\\n" > "$SVC_FILE"
				            fi
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
				      if [ "$IMAGE_TYPE" = "migrate" ]; then
				          yq -i 'select(.kind == "Job") .spec.template.spec.containers[0].image = env(IMAGE_URL)' "$FILE"
				          # Force ArgoCD out-of-sync by adding an annotation to the Deployment
				          if [ -f "$MANIFEST_PATH/manifest.yaml" ]; then
				              yq -i 'select(.kind == "Deployment") .metadata.annotations["helios.dev/last-migration"] = env(IMAGE_URL)' "$MANIFEST_PATH/manifest.yaml"
				          elif [ -f "$MANIFEST_PATH/deployment.yaml" ]; then
				              yq -i 'select(.kind == "Deployment") .metadata.annotations["helios.dev/last-migration"] = env(IMAGE_URL)' "$MANIFEST_PATH/deployment.yaml"
				          fi
				      else
				          yq -i 'select(.kind == "Deployment") .spec.template.spec.containers[].image = env(IMAGE_URL)' "$FILE"
				          yq -i 'select(.kind == "Deployment") .spec.replicas = env(REPLICAS)' "$FILE"
				          yq -i 'select(.kind == "Deployment") .spec.template.spec.containers[].ports[0].containerPort = env(PORT)' "$FILE"
				          yq -i 'select(.kind == "Service") .spec.ports[0].port = env(PORT)' "$FILE"
				          yq -i 'select(.kind == "Service") .spec.ports[0].targetPort = env(PORT)' "$FILE"
				      fi
				  fi
				done
				"""
		}, {
			// Step 3: Commit and Push using Git image
			name:  "git-commit"
			image: _config.images.gitClone
			envFrom: [{
				secretRef: {
					// Use task param instead of hardcoded secret so environments can vary.
					name:     "$(params.gitops-secret-ref)"
					// Keep step runnable even when secret is absent; script handles missing creds.
					optional: true
				}
			}]
			script: """
				#!/bin/sh
				set -e
					if ! command -v git >/dev/null 2>&1; then
						apk add --no-cache git
					fi
				WORKSPACE_PATH="$(workspaces.gitops-repo.path)"
				cd "$WORKSPACE_PATH"
				git_safe() {
					git -c safe.directory="$WORKSPACE_PATH" "$@"
				}

				git_safe config user.email "$(params.gitops-author-email)"
				git_safe config user.name "$(params.gitops-author-name)"

				if [ -n "${username:-}" ] && [ -n "${password:-}" ]; then
				  RAW_URL="$(params.gitops-repo-url)"
				  # Strip embedded credentials without sed (minimal images may not include it).
				  case "$RAW_URL" in
				    https://*@*)
				      RAW_URL="https://${RAW_URL#https://*@}"
				      ;;
				    http://*@*)
				      RAW_URL="http://${RAW_URL#http://*@}"
				      ;;
				  esac
				  case "$RAW_URL" in
				    https://*)
				      REPO_URL_WITH_AUTH="https://${username}:${password}@${RAW_URL#https://}"
				      ;;
				    http://*)
				      REPO_URL_WITH_AUTH="http://${username}:${password}@${RAW_URL#http://}"
				      ;;
				    *)
				      echo "Unsupported GitOps repo URL scheme: $RAW_URL"
				      exit 1
				      ;;
				  esac
				  git_safe remote set-url origin "${REPO_URL_WITH_AUTH}"
				else
				    echo "WARNING: username or password env vars not set. Push might fail."
				fi

				git_safe add .
				if git_safe diff-index --quiet HEAD --; then
				    echo "No changes to commit"
				else
				    git_safe commit -m "chore: Update image=$(params.new-image-url) [skip-ci]"
				    git_safe push origin "$(params.gitops-repo-branch)"
				fi
				"""
		}]
	}
}
