package traits

import "helios.io/cue/definitions/bases"

// ServiceTrait - Expose Service cho Component

#ServiceTrait: {
	parameter: {
		name:       string
		port:       int
		targetPort: int | *port
	}

	_p: parameter

	outputs: {
		service: (bases.#Service & {
			parameter: {
				name:       _p.name
				port:       _p.port
				targetPort: _p.targetPort
			}
		}).output
	}
}
