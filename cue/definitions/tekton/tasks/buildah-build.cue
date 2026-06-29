package tasks

import "helios.io/cue/definitions/tekton"

// Buildah Build Task
#BuildahBuild: tekton.#TektonTask & {
	parameter: {
		name: string | *"buildah-build"
	}

	// Alias config for internal use
	_config: tekton.#Defaults

	output: spec: {
		params: [
			tekton.#CommonParams.image.name,
			tekton.#CommonParams.image.dockerfile,
			tekton.#CommonParams.image.contextSubpath,
			tekton.#CommonParams.image.dockerSecret,
			tekton.#CommonParams.image.storageDriver,
			tekton.#CommonParams.image.buildahIsolation,
			tekton.#CommonParams.image.buildPlatforms,
		]
		workspaces: [{
			name: "source"
		}]
		results: [{
			name: "IMAGE_URL"
		}]
		steps: [{
			name:  "build-and-push"
			image: "gcr.io/kaniko-project/executor:v1.23.2-debug"
			env: [{
				name: "GODEBUG"
				value: "http2client=0"
			}]
			script: """
				#!/bin/sh
				set -e

				CONTEXT_PATH="$(workspaces.source.path)/$(params.context-subpath)"
				DOCKERFILE_PATH="$CONTEXT_PATH/$(params.dockerfile)"

				# Kaniko supports one custom platform; use the first entry if a list is provided.
				PLATFORM_FLAG=""
				if [ -n "$(params.build-platforms)" ]; then
				  FIRST_PLATFORM=$(echo "$(params.build-platforms)" | cut -d',' -f1)
				  PLATFORM_FLAG="--custom-platform ${FIRST_PLATFORM}"
				fi

				/kaniko/executor --dockerfile "$DOCKERFILE_PATH" --context "$CONTEXT_PATH" --destination "$(params.image)" --digest-file /tekton/results/IMAGE_DIGEST $PLATFORM_FLAG
				"""
			volumeMounts: [{
				name:      "docker-config"
				mountPath: "/kaniko/.docker"
			}]
		}, {
			name:  "write-image-url"
			image: _config.images.alpine
			script: """
				#!/bin/sh
				set -e
				echo "$(params.image)@$(cat /tekton/results/IMAGE_DIGEST)" > $(results.IMAGE_URL.path)
				"""
		}]
		volumes: [{
			name: "docker-config"
			secret: {
				secretName: "$(params.docker-secret)"
				items: [{
					key:  ".dockerconfigjson"
					path: "config.json"
				}]
			}
		}]
	}
}
