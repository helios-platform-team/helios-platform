package definitions

// Definition for the Service
service: {
	// 1. INPUTS: Define what parameters the user must provide
	parameter: {
		name:       string
		port:       int | *80      // Default to port 80 if not provided
		targetPort: int | *8080    // Default to 8080 if not provided
	}

	// 2. OUTPUT: The actual Kubernetes Service object
	output: {
		apiVersion: "v1"
		kind:       "Service"
		metadata: {
			name: parameter.name
			labels: {
				// This label helps organize resources
				app: parameter.name 
			}
		}
		spec: {
			type: "ClusterIP" // As requested in the requirements
			
			// THE CONTRACT: This selector must match the Deployment's labels
			selector: {
				app: parameter.name
			}

			ports: [{
				name:       "http"
				port:       parameter.port
				targetPort: parameter.targetPort
				protocol:   "TCP"
			}]
		}
	}
}