package examples

import "helios.io/cue/definitions/schema"

// Example: Simple Web Application
// Dùng để test CUE engine và làm tham khảo

simpleApp: schema.#HeliosApp & {
	app: {
		name:        "my-app"
		namespace:   "default"
		owner:       "dev-team"
		description: "A simple web application example"

		components: [{
			name: "api-server"
			type: "web-service"
			properties: {
				image:    "myregistry/api:v1.0.0"
				port:     3000
				replicas: 2
				env: [{
					name:  "NODE_ENV"
					value: "production"
				}]
			}
			traits: [{
				type: "service"
				properties: {
					port: 3000
				}
			}, {
				type: "ingress"
				properties: {
					host: "api.example.com"
					port: 3000
				}
			}]
		}]
	}
}
