package main

// Import the folders
import "helios.io/templates/definitions/components"

#ComponentRegistry: {
    "web-service": components.#WebService
}

app: {
    name: "my-platform-app"
    components: [...]
}

// 3. ENGINE: Generates the final Kubernetes Objects
kubernetesObjects: [
    // 1. First Loop: Iterate over user components
    for comp in app.components
    // 2. Define variables (Look, NO curly braces opening here!)
    let CompDef = #ComponentRegistry[comp.type]
    let RenderedComp = CompDef & {
        parameter: comp.properties
        parameter: name: comp.name
    }
    // 3. Second Loop: Iterate over the outputs (Deployment, Service)
    for resourceName, resourceBody in RenderedComp.outputs {
        // 4. Final Output: This generates one list item per resource
        resourceBody.output
    }
]