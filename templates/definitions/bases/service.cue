package bases

#Service: {
    parameter: {
        name:       string
        port:       int
        targetPort: int
    }

    output: {
        apiVersion: "v1"
        kind:       "Service"
        metadata: {
            "name": parameter.name
            labels: app: parameter.name
        }
        spec: {
            type: "ClusterIP"
            selector: app: parameter.name
            ports: [{
                "port":       parameter.port
                "targetPort": parameter.targetPort
                protocol:     "TCP"
            }]
        }
    }
}