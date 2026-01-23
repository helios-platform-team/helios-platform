package components

import "helios.io/templates/definitions/bases"

#WebService: {
    parameter: {
        name:       string
        image:      string
        replicas:   int | *1
        port:       int | *8080
        exposePort: int | *80
    }

    outputs: {
        // USE THE BASE HERE instead of typing it all out
        deployment: bases.#Deployment & {
            name:     parameter.name
            image:    parameter.image
            replicas: parameter.replicas
            port:     parameter.port
        }

        if parameter.exposePort != _|_ {
            // USE THE BASE HERE
            service: bases.#Service & {
                name:       parameter.name
                port:       parameter.exposePort
                targetPort: parameter.port
            }
        }
    }
}