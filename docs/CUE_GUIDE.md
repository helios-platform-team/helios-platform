# Deploying a New Application (Updated)

This guide explains how to deploy a new application using the current CUE-based system architecture, which is built around an `app` definition containing multiple components.

---

## 1. Create a Service Definition File

To deploy a new application, create a `.cue` file inside the `templates/services/` directory.

Instead of defining a single workload directly, the system now uses an **application structure (`app`)** that contains a list of **components**.

### Example

**File:** `templates/services/myApp.cue`

```cue
package main

// The user defines application components here.
// The system engine handles rendering and resource generation.
app: {
    name: "my-application-name"
    components: [
        {
            name: "backend"
            type: "web-service" // Must match a key in the ComponentRegistry
            properties: {
                image:      "my-backend-image:v1"
                replicas:   3      // Optional, default is 1
                port:       8080   // Container port, default is 8080
                exposePort: 80     // Service port. If set, a Service is created.
            }
        }
    ]
}
```

## 2. Main Execution Commands

Below are the main commands used when working with the project.

| Command                                                                                                    | Description                                             |
| ---------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `cue vet ./templates/...`                                                                                  | Validate logic and data types across the entire project |
| `cue export ./templates/system/builder.cue ./templates/services/myApp.cue -e kubernetesObjects --out yaml` | Render the application into standard Kubernetes YAML    |

```

```
