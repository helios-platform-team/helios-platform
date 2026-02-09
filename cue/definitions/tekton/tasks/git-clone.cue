package tasks

import (
	"helios.io/cue/definitions/tekton/bases"
)

#GitClone: bases.#TektonTask & {
	metadata: name: "git-clone"
	spec: {
		params: [
			bases.#CommonParams.git_url,
			bases.#CommonParams.git_revision,
		]
		workspaces: [{
			name:        "output"
			description: "The workspace where the source code will be cloned."
		}]
		steps: [{
			name:  "clone"
			image: "alpine/git:latest"
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
