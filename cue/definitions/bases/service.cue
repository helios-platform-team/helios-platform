package bases

// Base renderer cho Kubernetes Service
// Pure parameter → output transformation

#Service: {
	parameter: {
		name:       string
		port:       int
		targetPort: int | *port
	}

	output: {
		apiVersion: "v1"
		kind:       "Service"
		metadata: {
			name: parameter.name
			labels: {
				app:                      parameter.name
				"app.kubernetes.io/name": parameter.name
			}
		}
		spec: {
			selector: {
				app:                      parameter.name
				"app.kubernetes.io/name": parameter.name
			}
			ports: [{
				port:       parameter.port
				targetPort: parameter.targetPort
			}]
			type: "ClusterIP"
		}
	}
}
