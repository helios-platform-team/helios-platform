package traits

import "helios.io/templates/definitions/bases"

#ServiceTrait: {
    let Params = parameter

    parameter: {
        name:          string
        componentName: string 
        port:          int
        targetPort:    int
    }

    outputs: {
        service: bases.#Service & {
            parameter: {
                name:          Params.name
                componentName: Params.componentName 
                port:          Params.port
                targetPort:    Params.targetPort
            }
        }
    }
}