package bases

#Deployment: {
    name:     string
    image:    string
    replicas: int
    port:     int
    
    output: {
        apiVersion: "apps/v1"
        kind:       "Deployment"
        metadata: "name": name
        spec: {
            "replicas": replicas
            selector: matchLabels: app: name
            template: {
                metadata: labels: app: name
                spec: containers: [{
                    "name":  name
                    "image": image
                    ports: [{containerPort: port}]
                }]
            }
        }
    }
}