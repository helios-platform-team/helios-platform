package main

// The user just lists components. The engine handles the rest.
app: {
    components: [
        {
            name: "backend"
            type: "web-service" // Matches registry key
            properties: {
                image:      "my-backend:v1"
                replicas:   3
                port:       8080
                exposePort: 80
            }
        },
        {
            name: "frontend"
            type: "web-service"
            properties: {
                image:      "my-frontend:v1"
                port:       3000
                // Note: exposePort defaults to 80
            }
        }
    ]
}