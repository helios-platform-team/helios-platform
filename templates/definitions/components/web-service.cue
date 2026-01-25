package components

import "helios.io/templates/definitions/bases"

#WebService: {
    parameter: {
        name:       string
        image:      string
        replicas:   int | *1
        port:       int | *8080
    }

    outputs: {
        deployment: bases.#Deployment & {
            parameter: {
                name:     parameter.name
                image:    parameter.image
                replicas: parameter.replicas
                port:     parameter.port
            }
        }
    }
}