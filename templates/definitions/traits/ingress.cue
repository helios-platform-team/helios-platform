package traits

import "helios.io/templates/definitions/bases"

#IngressTrait: {
    let Params = parameter

    parameter: {
        name:        string
        host:        string
        path:        string | *"/"
        serviceName: string
        servicePort: int
    }

    outputs: {
        ingress: bases.#Ingress & {
            parameter: {
                name:        Params.name
                host:        Params.host
                path:        Params.path
                serviceName: Params.serviceName
                servicePort: Params.servicePort
            }
        }
    }
}