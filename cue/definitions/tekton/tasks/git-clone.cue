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
				
				# Clean the workspace if it exists
				echo "Cleaning workspace: $(workspaces.output.path)"
				rm -rf $(workspaces.output.path)/*
				rm -rf $(workspaces.output.path)/.[!.]*
				
				# Transform localhost URLs for in-cluster access
				GIT_URL="$(params.url)"
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
				
				# Clone the repository
				echo "Cloning $GIT_URL to $(workspaces.output.path)"
				git clone "$GIT_URL" $(workspaces.output.path)
				
				# Checkout the specified revision
				cd $(workspaces.output.path)
				echo "Checking out $(params.revision)"
				git checkout $(params.revision)
				
				echo "Git clone completed successfully"
				"""
		}]
	}
}
