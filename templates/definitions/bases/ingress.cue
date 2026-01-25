package bases

#Ingress: {
	parameter: {
		name:        string
		host:        string
		path:        string
		serviceName: string
		servicePort: int
	}

	output: {
		apiVersion: "networking.k8s.io/v1"
		kind:       "Ingress"
		metadata: name: parameter.name
		spec: rules: [{
			host: parameter.host
			http: paths: [{
				path:     parameter.path
				pathType: "Prefix"
				backend: service: {
					name: parameter.serviceName
					port: number: parameter.servicePort
				}
			}]
		}]
	}
}