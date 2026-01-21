package definitions

#Service: {
	name:       string
	port:       int
	targetPort: int

	output: {
		apiVersion: "v1"
		kind:       "Service"
		metadata: {
			"name": name
			labels: app: name
		}
		spec: {
			type: "ClusterIP"
			selector: app: name
			ports: [{
				"port":       port
				"targetPort": targetPort
				protocol:   "TCP"
			}]
		}
	}
}