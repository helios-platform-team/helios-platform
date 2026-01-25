package traits

import "helios.io/templates/definitions/bases"

#AutoscaleTrait: {
	parameter: {
		name:        string
		minReplicas: int | *1
		maxReplicas: int | *10
		cpuPercent:  int | *50
	}

	outputs: {
		scaler: bases.#HorizontalPodAutoscaler & {
			parameter: {
				name:        parameter.name
				minReplicas: parameter.minReplicas
				maxReplicas: parameter.maxReplicas
				cpuPercent:  parameter.cpuPercent
			}
		}
	}
}