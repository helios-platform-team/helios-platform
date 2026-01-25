package traits

import "helios.io/templates/definitions/bases"

#IngressTrait: {
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
				name:        parameter.name
				host:        parameter.host
				path:        parameter.path
				serviceName: parameter.serviceName
				servicePort: parameter.servicePort
			}
		}
	}
}