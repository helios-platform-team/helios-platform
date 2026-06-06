package tasks

import "helios.io/cue/definitions/tekton"

// Git Clone Task
#GitClone: tekton.#TektonTask & {
	parameter: {
		name: "git-clone"
	}

	// Alias config for internal use
	_config: tekton.#Defaults

	output: spec: {
		params: [
			tekton.#CommonParams.git.url,
			tekton.#CommonParams.git.revision,
		]
		workspaces: [{
			name:        "output"
			description: "The workspace where the source code will be cloned."
		}]
		steps: [{
			name:  "clone"
			image: _config.images.gitClone
			script: """
				#!/bin/sh
				set -e
				if ! command -v git >/dev/null 2>&1; then
					apk add --no-cache git
				fi

				# Transform localhost URLs for in-cluster access
				GIT_URL="$(params.url)"
				WORKSPACE_PATH="$(workspaces.output.path)"

				# Tekton PVCs can be owned by a different UID/GID than the step user.
				# Force git to trust this workspace path to avoid "dubious ownership".
				git_safe() {
					git -c safe.directory="$WORKSPACE_PATH" "$@"
				}

				case "$GIT_URL" in
					*localhost:3030*)
						echo "Transforming localhost URL to in-cluster address"
						GIT_URL=$(echo "$GIT_URL" | sed 's|http://localhost:3030/|http://gitea-http.gitea.svc.cluster.local:3000/|g')
						echo "Transformed URL: $GIT_URL"
						;;
					*)
						echo "Using URL as-is: $GIT_URL"
						;;
				esac
				
				# Prepare workspace without relying on external rm binary.
				# Some minimal git images used in Tekton don't include coreutils.
				cd "$WORKSPACE_PATH"
				echo "Preparing workspace with git primitives: $WORKSPACE_PATH"
				if [ -d .git ]; then
					git_safe remote remove origin || true
				else
					git_safe init
				fi
				
				# Fetch and checkout the specified revision
				git_safe remote add origin "$GIT_URL"
				git_safe fetch --depth=1 origin "$(params.revision)"
				git_safe checkout -B helios-build FETCH_HEAD
				echo "Checking out $(params.revision)"
				git_safe reset --hard FETCH_HEAD
				git_safe clean -fdx
				
				echo "Git clone completed successfully"
				"""
		}]
	}
}
