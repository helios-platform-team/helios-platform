package traits

import "helios.io/templates/definitions/bases"

#AutoscaleTrait: {
    let Params = parameter

    parameter: {
        name:        string
        minReplicas: int | *1
        maxReplicas: int | *10
        cpuPercent:  int | *50
    }

    outputs: {
        scaler: bases.#HorizontalPodAutoscaler & {
            parameter: {
                name:        Params.name
                minReplicas: Params.minReplicas
                maxReplicas: Params.maxReplicas
                cpuPercent:  Params.cpuPercent
            }
        }
    }
}