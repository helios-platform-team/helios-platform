package traits

import "helios.io/templates/definitions/bases"

#ServiceTrait: {
	parameter: {
		name:       string
		port:       int
		targetPort: int
	}

	outputs: {
		service: bases.#Service & {
			parameter: {
				name:       parameter.name
				port:       parameter.port
				targetPort: parameter.targetPort
			}
		}
	}
}