package schema

// Application Model - contract giữa Operator và CUE Engine
// KHÔNG mirror CRD YAML, chỉ chứa Application Model thuần tuý

#HeliosApp: {
	app: {
		name:         string
		namespace:    string
		owner?:       string
		description?: string

		components: [...#Component]
	}
}

#Component: {
	name:       string
	type:       string
	properties: {...}
	traits?: [...#Trait]
}

#Trait: {
	type:       string
	properties: {...}
}
