# Deploying a New Application

## 1. Create a Service Definition File

To deploy a new application, create a `.cue` file inside the `templates/services/` directory with minimal content as shown below:

```cue
package services

import "helios.io/templates/definitions"

myAppName: definitions.#Workload & {
    name:     "application-name"
    image:    "docker-image-path:tag"
    replicas: 3
    port:     80
}
```

- name: The name of the application

- image: Docker image path with tag

- replicas: Number of running replicas

- port: Exposed container port

## 2. Main Execution Commands

Below are the main commands used when working with the project:
| Command | Description |
| ------------- |:-------------:|
| `cue vet ./templates/...` | Validate logic and data types across the entire project |
| `cue export ./templates/definitions/... ./templates/services/myApp.cue -e 'myApp.objects' --out yaml` | Render a specific application into YAML |
