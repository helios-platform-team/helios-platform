package main

app: {
    name: "guestbook"
    components: [
        {
            name: "frontend"
            type: "web-service"
            properties: {
                image:    "my-frontend:v1"
                replicas: 2
                port:     8080
            }
            traits: [
                {
                    type: "service"
                    properties: {
                        name:       "frontend-svc"
                        port:       80
                        targetPort: 8080
                    }
                },
                {
                    type: "ingress"
                    properties: {
                        name:        "frontend-ingress"
                        host:        "guestbook.example.com"
                        serviceName: "frontend-svc"
                        servicePort: 80
                    }
                }
            ]
        },
        {
            name: "backend"
            type: "web-service"
            properties: {
                image:    "my-backend:v1"
                replicas: 1
                port:     5000
            }
            // Backend has no ingress, only internal service
            traits: [
                {
                    type: "service"
                    properties: {
                        name:       "backend-svc"
                        port:       5000
                        targetPort: 5000
                    }
                }
            ]
        }
    ]
}