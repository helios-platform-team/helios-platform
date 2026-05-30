package bases

// Base renderer cho Kubernetes Deployment
// Pure parameter → output transformation

#Deployment: {
	parameter: {
		name:     string
		image:    string
		replicas: int | *1
		port:     int
		
		// FIX: Allow both standard 'value' strings and Kubernetes 'valueFrom' objects
		env: [...{
			name: string
			value?: string
			valueFrom?: {...}
		}] | *[]

		initContainers?: [...{
			name: string
			image: string
			command?: [...string]
			env?: [...{
				name: string
				value?: string
				valueFrom?: {...}
			}]
		}]
	}

	output: {
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: {
			name: parameter.name
			labels: {
				app:                    parameter.name
				"helios.io/managed-by": "operator"
			}
		}
		spec: {
			replicas: parameter.replicas
			revisionHistoryLimit: 2
			selector: matchLabels: app: parameter.name
			template: {
				metadata: labels: app: parameter.name
				spec: {
					if parameter.initContainers != _|_ {
						initContainers: parameter.initContainers
					}
					containers: [{
						name:  parameter.name
						image: parameter.image
						ports: [{containerPort: parameter.port}]
						if len(parameter.env) > 0 {
							env: parameter.env
						}
						resources: {
							requests: {cpu: "100m", memory: "128Mi"}
							limits: {cpu: "500m", memory: "512Mi"}
						}
					}]
				}
			}
		}
	}
}