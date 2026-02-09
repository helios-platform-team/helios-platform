package tekton

// Kaniko Build Task
#KanikoBuild: #TektonTask & {
	parameter: {
		name: "kaniko-build"
	}

	// Alias config for internal use
	_config: #Defaults

	output: spec: {
		params: [
			#CommonParams.image.name,
			#CommonParams.image.dockerfile,
			#CommonParams.image.contextSubpath,
			#CommonParams.image.dockerSecret,
		]
		workspaces: [{
			name: "source"
		}]
		results: [{
			name: "IMAGE_URL"
		}]
		steps: [{
			name:  "build-and-push"
			image: _config.images.kaniko
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
			image: _config.images.alpine
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
