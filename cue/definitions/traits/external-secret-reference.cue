package traits

import "strings"

// ExternalSecretReferenceTrait — mount an existing Kubernetes Secret into the
// workload via Deployment envFrom (all keys become environment variables).
//
// The Secret itself is not rendered here; create it via GitOps (e.g. sealed-secrets),
// ExternalSecrets, or your platform tooling. Declaring this trait in the HeliosApp
// Git manifest keeps wiring auditable and lets the CUE engine emit envFrom.
//
// Usage:
//   traits: [{
//       type: "external-secret-reference"
//       properties: secretName: "my-service-db"
//   }]
//
// Injection into Deployments is applied in engine/builder.cue for components that
// support envFromSecrets (currently web-service).

#ExternalSecretReferenceTrait: {
	parameter: {
		name!: string & strings.MinRunes(1)

		// Kubernetes Secret name (full name, including any service prefix).
		secretName!: string & strings.MinRunes(1)
	}

	_p: parameter

	// No standalone Kubernetes object; builder merges secretName into Deployment envFrom.
	outputs: {}
}
