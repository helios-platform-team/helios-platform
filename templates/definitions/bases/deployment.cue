package bases

#Deployment: {
    parameter: {
        name:     string
        image:    string
        replicas: int | *1
        port:     int
    }

    output: {
        apiVersion: "apps/v1"
        kind:       "Deployment"
        metadata: "name": parameter.name
        spec: {
            "replicas": parameter.replicas
            selector: matchLabels: app: parameter.name
            template: {
                metadata: labels: app: parameter.name
                spec: containers: [{
                    "name":  parameter.name
                    "image": parameter.image
                    ports: [{containerPort: parameter.port}]
                }]
            }
        }
    }
}