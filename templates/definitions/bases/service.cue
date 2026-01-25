package bases

#Service: {
    parameter: {
        name:          string
        componentName: string 
        port:          int
        targetPort:    int
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
            selector: app: parameter.componentName 
            ports: [{
                "port":       parameter.port
                "targetPort": parameter.targetPort
                protocol:     "TCP"
            }]
        }
    }
}