package definitions

#Workload: {
    name:     string
    image:    string
    replicas: int | *2
    port:     int | *80
    targetPort: int | *8080

    k8sDeployment: #Deployment & {
        "name":     name
        "image":    image
        "replicas": replicas
        "port":     targetPort
    }

    k8sService: #Service & {
        "name":       name
        "port":       port
        "targetPort": targetPort
    }
}