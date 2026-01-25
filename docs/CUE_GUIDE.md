# Helios Architecture & Workflow

---

## 1. User Interaction (YAML)

Users **DO NOT** edit CUE files.  
Instead, users interact with Helios via standard Kubernetes **Custom Resource Definitions (CRDs)** written in YAML.

### Example User Input (`app.yaml`)

```yaml
apiVersion: helios.io/v1
kind: Application
metadata:
  name: guestbook
spec:
  components:
    - name: frontend
      type: web-service
      properties:
        image: my-frontend:v1
        port: 8080
      traits:
        - type: service
          properties:
            port: 80
```

## 2. Internal Engine (CUE)

The CUE files in this repository (`templates/`) form the **Execution Engine** of Helios.

The **Helios Operator** is responsible for:

- Reading the user-provided YAML (Application CRD)
- Converting the YAML into a CUE value
- Executing the CUE engine to generate low-level Kubernetes resources

### Directory Structure

#### `templates/definitions/`

Contains definitions for **Components** (e.g. `WebService`) and **Traits** (e.g. `Ingress`, `Service`).  
These definitions act like _classes_ in the system, describing reusable behavior and structure.

#### `templates/system/`

Contains `builder.cue`, which acts as the **main function** of the engine.  
It iterates over components and their traits to render the final outputs.

#### `examples/`

Contains CUE files that **simulate user input**.  
These files are used for **testing and development only** and are **not intended for production use**.

## 3. Workflow

1. The user applies an `Application` YAML to the Kubernetes cluster.
2. The Helios Operator detects the change in the custom resource.
3. The operator maps the YAML `spec` into a CUE value:

   ```cue
   app: { ... }
   ```

4. The CUE engine (`builder.cue`) evaluates the input data against the component and trait definitions to produce concrete Kubernetes manifests.

5. The Helios Operator then applies the generated Kubernetes objects  
   (e.g. `Deployment`, `Service`, `Ingress`) to the target cluster.

## 4. Main Execution Commands

Below are the main commands used when working with the project.

| Command                                                                                               | Description                                             |
| ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `cue vet ./templates/...`                                                                             | Validate logic and data types across the entire project |
| `cue export ./examples/simple-app.cue ./templates/system/builder.cue -e kubernetesObjects --out yaml` | Render the application into standard Kubernetes YAML    |
