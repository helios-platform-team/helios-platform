package components

import "helios.io/cue/definitions/bases"

// WebService Component - OAM compliant
// Chỉ sinh Deployment, KHÔNG sinh Service (Service là Trait)

#WebService: {
	// Input parameters từ CRD
	parameter: {
		name:     string
		image:    string
		replicas: int | *1
		port:     int | *8080
		env: [...{name: string, value: string}] | *[]
	}

	// Alias để tránh scope issues
	_p: parameter

	// Output kubernetes resources
	outputs: {
		deployment: (bases.#Deployment & {
			parameter: {
				name:     _p.name
				image:    _p.image
				replicas: _p.replicas
				port:     _p.port
				env:      _p.env
			}
		}).output
	}
}
