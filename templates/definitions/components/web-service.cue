package components

import "helios.io/templates/definitions/bases"

#WebService: {
    // Alias the top-level parameter to avoid shadowing
    let Params = parameter

    parameter: {
        name:       string
        image:      string
        replicas:   int | *1
        port:       int | *8080
    }

    outputs: {
        deployment: bases.#Deployment & {
            parameter: {
                // Use the alias here
                name:     Params.name
                image:    Params.image
                replicas: Params.replicas
                port:     Params.port
            }
        }
    }
}