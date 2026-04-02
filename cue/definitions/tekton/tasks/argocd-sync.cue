package tasks

import "helios.io/cue/definitions/tekton"

// ArgoCDSync requests an Argo CD sync by patching the Application CR (no Argo CD API token required).
// Requires RBAC: get,patch on applications.argoproj.io for that Application (Helios operator provisions a Role + RoleBinding).
#ArgoCDSync: tekton.#TektonTask & {
	parameter: {
		name: "argocd-sync"
	}

	_config: tekton.#Defaults

	output: spec: {
		params: [
			tekton.#CommonParams.argocd.namespace,
			tekton.#CommonParams.argocd.appName,
		]
		steps: [{
			name:  "argocd-sync"
			image: _config.images.kubectl
			script: """
				#!/bin/sh
				set -e
				NS="$(params.argocd-namespace)"
				APP="$(params.argocd-app-name)"
				echo "Requesting Argo CD sync: application/$APP in namespace $NS"
				kubectl patch application "$APP" -n "$NS" --type merge \\
				  -p '{"operation":{"initiatedBy":{"username":"tekton"},"sync":{"prune":true,"syncStrategy":{"hook":{}}}}}'
				echo "Argo CD sync operation submitted."
				"""
		}]
	}
}
