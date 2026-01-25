# Helios CUE Engine

This directory contains the [CUE](https://cuelang.org/) definitions and rendering logic for the Helios Platform. It serves as the "brainhead" of the Helios Operator, responsible for transforming high-level Helios application models into low-level Kubernetes resources.

## 📂 Directory Structure

- **`cue.mod/`**: Standard CUE module configuration and dependencies.
- **`definitions/`**:
  - **`schema/`**: Core API schemas. `app.cue` defines the `#HeliosApp` contract between the Operator and the Engine.
  - **`bases/`**: Base Kubernetes object templates (Deployment, Service, etc.) used by components and traits.
  - **`components/`**: Definitions for application components (e.g., `web-service`).
  - **`traits/`**: Definitions for operational behaviors (e.g., `ingress`, `service` exposure).
- **`engine/`**: The core rendering logic. `builder.cue` implements the generic rendering loop that processes components and traits.
- **`examples/`**: Reference application definitions used for documentation and testing.

## 🚀 How It Works

1. **Input**: The Operator passes a `HeliosApp` object (JSON) into the CUE engine.
2. **Registry**: The `engine/builder.cue` looks up the component and trait types in its internal registries.
3. **Rendering**: The engine iterates through the requested components and traits, merging user properties with the predefined CUE templates.
4. **Output**: The final result is a list of Kubernetes manifests (`kubernetesObjects`) ready to be applied to the cluster.

## 🧪 How to Test

### 1. Using CUE CLI

If the `cue` command is not in your PATH, it is typically located at `~/go/bin/cue`. You can substitute `cue` with `~/go/bin/cue` in the commands below if needed.

**Evaluate the rendered output for an example:**

```bash
# From the /cue directory
printf 'package engine\nimport "helios.io/cue/examples"\ninput: examples.simpleApp' > engine/test.cue && ~/go/bin/cue eval ./engine -e kubernetesObjects --out yaml && rm engine/test.cue
```

**Validate an application against the schema:**

```bash
# From the /cue directory
~/go/bin/cue vet ./definitions/schema ./examples/simple-app.cue
```

### 2. Using Go Tests

The primary integration tests reside in the Helios Operator codebase. You can run them from the **project root** using the `-C` flag to specify the module directory:

```bash
# From the project root (helios-platform/)
go -C apps/operator test -v ./internal/cue/...
```

## 🛠 Adding New Features

- **To add a new component**: Create a new file in `definitions/components/` and register it in `engine/builder.cue` under `#ComponentRegistry`.
- **To add a new trait**: Create a new file in `definitions/traits/` and register it in `engine/builder.cue` under `#TraitRegistry`.
- **To update K8s base templates**: Modify the files in `definitions/bases/`.
