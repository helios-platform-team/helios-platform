package schema

// Defines the standard structure for the entire Application
#HeliosApp: {
    name: string

    // List of components is mandatory
    components: [...#Component]
}

// Defines the structure of a Component
#Component: {
    name: string
    type: string

    // Properties is a flexible object (map) containing image, replicas, etc.
    properties: {...}

    // Traits is an optional list (may or may not be present)
    traits?: [...#Trait]
}

// Defines the structure of a Trait
#Trait: {
    type: string
    properties: {...}
}