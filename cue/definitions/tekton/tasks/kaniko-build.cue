package tasks

import (
	"helios.io/cue/definitions/tekton/bases"
)

#KanikoBuild: bases.#TektonTask & {
	metadata: name: "kaniko-build"
	spec: {
		params: [{
			name:        "IMAGE"
			description: "Name (reference) of the image to build."
			type:        "string"
		}, {
			name:    "DOCKERFILE"
			default: "Dockerfile"
			type:    "string"
		}, {
			name:        "CONTEXT_SUBPATH"
			description: "Subdirectory within the workspace where the Dockerfile is located"
			default:     ""
			type:        "string"
		}, {
			name:        "docker-secret"
			description: "Name of the secret containing docker credentials"
			default:     "docker-credentials"
			type:        "string"
		}]
		workspaces: [{
			name: "source"
		}]
		results: [{
			name: "IMAGE_URL"
		}]
		steps: [{
			name:  "build-and-push"
			image: "gcr.io/kaniko-project/executor:latest"
			env: [{
				name:  "DOCKER_CONFIG"
				value: "/kaniko/.docker"
			}]
			command: ["/kaniko/executor"]
			args: [
				"--dockerfile=$(params.DOCKERFILE)",
				"--context=$(workspaces.source.path)/$(params.CONTEXT_SUBPATH)",
				"--destination=$(params.IMAGE)",
				"--digest-file=/tekton/results/IMAGE_DIGEST",
			]
			volumeMounts: [{
				name:      "docker-config"
				mountPath: "/kaniko/.docker"
			}]
		}, {
			name:  "write-image-url"
			image: "alpine:latest"
			script: """
				#!/bin/sh
				set -e
				echo "$(params.IMAGE)@$(cat /tekton/results/IMAGE_DIGEST)" > $(results.IMAGE_URL.path)
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
