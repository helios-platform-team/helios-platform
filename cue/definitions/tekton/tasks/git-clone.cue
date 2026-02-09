package tekton

// Git Clone Task
#GitClone: #TektonTask & {
	parameter: {
		name: "git-clone"
	}

	// Alias config for internal use
	_config: #Defaults

	output: spec: {
		params: [
			#CommonParams.git.url,
			#CommonParams.git.revision,
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
				
				# Clone the repository
				echo "Cloning $(params.url) to $(workspaces.output.path)"
				git clone $(params.url) $(workspaces.output.path)
				
				# Checkout the specified revision
				cd $(workspaces.output.path)
				echo "Checking out $(params.revision)"
				git checkout $(params.revision)
				
				echo "Git clone completed successfully"
				"""
		}]
	}
}
