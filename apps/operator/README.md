# Helios Operator

## Description
The Helios Operator is a Kubernetes controller designed for the Helios Developer Platform. It reconciles `HeliosApp` custom resources to coordinate GitOps repository structures, configure backing databases, deploy Tekton CI pipelines, and orchestrate ArgoCD deployment applications.

## Getting Started
See [GIT_OPS_GUIDE.md](GIT_OPS_GUIDE.md) for detailed GitOps setup instructions.

### Prerequisites
The following tools are required for development (managed via `mise` using the project's local `mise.toml` configuration):
- Go: `1.26.3`
- Node.js: `22`
- Yarn: `4.15.0`
- CUE: `0.16.1`
- Make: `4.4.1`
- Operator SDK: `1.42.2`
- Kubectl: `1.36.1+`
- K3d: `5.8.3+`
- Docker: `29.0.0+` (Running daemon)

### E2E Testing
To run the end-to-end integration tests locally on an isolated, temporary Kubernetes cluster:
```sh
make test-e2e
```
This target installs k3d if needed, provisions an ephemeral cluster, builds and loads the operator docker image, deploys Cert-Manager, runs Ginkgo integration tests, and automatically cleans up the cluster afterward.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/operator:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/operator/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
operator-sdk edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request. Make sure all formatting, linting, and tests (`make lint`, `make test`, and `make test-e2e`) pass successfully before submitting changes.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

