# 🚀 Helios Operator

> GitOps Automation Platform for Kubernetes

[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.34+-blue.svg)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> Fully automate your deployment workflow from code commit to production

[Documentation](#-documentation) • [Get Started](docs/GETTING_STARTED.md) • [Architecture](docs/ARCHITECTURE.md)

---

## 🎯 What is Helios?

**Helios Operator** is an intelligent Kubernetes Operator that helps you deploy applications using GitOps methodology automatically and easily. Just create a simple YAML file, and Helios handles the entire workflow!

### ✨ Key Features

- 🎯 **Simplified GitOps**: Single CRD, no complex configuration needed
- 🔄 **Fully Automated**: From manifest generation to deployment, zero manual intervention
- 📊 **Real-time Monitoring**: Track deployment status directly from Kubernetes
- 🛡️ **Production-ready**: RBAC, security, and best practices built-in
- 🎭 **Smart Reconciliation**: Only rebuild when spec changes, saving resources

**→ [Get Started with Helios](docs/GETTING_STARTED.md)** | **[View API Reference](docs/API_REFERENCE.md)**

---

## 📚 Documentation

Comprehensive documentation for using, developing, and maintaining Helios Operator.

### 🎯 Getting Started

| Document                                             | Description                                                         |
| ---------------------------------------------------- | ------------------------------------------------------------------- |
| **[Getting Started Guide](docs/GETTING_STARTED.md)** | Step-by-step tutorial: install, create first app, verify deployment |
| **[Setup Guide](docs/SETUP_GUIDE.md)**               | Detailed setup: cluster, dependencies, configuration                |
| **[Architecture](docs/ARCHITECTURE.md)**             | Understand 3-phase GitOps workflow and internal design              |

### 🔧 Development & Testing

| Document                                     | Description                                 |
| -------------------------------------------- | ------------------------------------------- |
| **[Development Guide](docs/DEVELOPMENT.md)** | Build, run locally, development workflow    |
| **[Testing Guide](docs/TESTING_GUIDE.md)**   | Unit tests, E2E tests, debugging strategies |
| **[API Reference](docs/API_REFERENCE.md)**   | Complete HeliosApp CRD specification        |

### 🚀 Production & Operations

| Document                                                   | Description                                  |
| ---------------------------------------------------------- | -------------------------------------------- |
| **[Production Deployment](docs/PRODUCTION_DEPLOYMENT.md)** | HA setup, security hardening, best practices |
| **[Monitoring](docs/MONITORING.md)**                       | Metrics, alerts, Grafana dashboards          |
| **[Troubleshooting](docs/TROUBLESHOOTING.md)**             | Common issues, debugging, solutions          |

→ **[Complete Documentation Index](docs/README.md)**

### 📖 Quick Navigation

| I want to...            | Documentation                                       |
| ----------------------- | --------------------------------------------------- |
| Deploy my first app     | → [Getting Started](docs/GETTING_STARTED.md)        |
| Understand architecture | → [Architecture Guide](docs/ARCHITECTURE.md)        |
| Check API specs         | → [API Reference](docs/API_REFERENCE.md)            |
| Fix an issue            | → [Troubleshooting](docs/TROUBLESHOOTING.md)        |
| Deploy to production    | → [Production Guide](docs/PRODUCTION_DEPLOYMENT.md) |
| Setup monitoring        | → [Monitoring Guide](docs/MONITORING.md)            |
| Develop locally         | → [Development Guide](docs/DEVELOPMENT.md)          |
| Run tests               | → [Testing Guide](docs/TESTING_GUIDE.md)            |

---

## 📊 Project Status

| Component          | Status      | Version |
| ------------------ | ----------- | ------- |
| Core Operator      | ✅ Stable   | v1.0.0  |
| Tekton Integration | ✅ Complete | -       |
| ArgoCD Integration | ✅ Complete | -       |
| E2E Tests          | ✅ Passing  | -       |
| Documentation      | ✅ Complete | -       |

### Technology Stack

#### Core

- [Kubebuilder](https://book.kubebuilder.io/) v4.9+ - Operator framework
- [Go](https://golang.org/) 1.25+ - Programming language
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controllers

#### GitOps Components

- [Tekton Pipelines](https://tekton.dev/) latest - CI/CD pipelines
- [Tekton Triggers](https://tekton.dev/docs/triggers/) latest - Event handling
- [ArgoCD](https://argo-cd.readthedocs.io/) latest - GitOps deployment

#### Testing

- [Ginkgo](https://onsi.github.io/ginkgo/) v2 - BDD testing framework
- [Gomega](https://onsi.github.io/gomega/) - Matcher library

### Future Enhancements

- Multi-cluster support
- Helm chart repository
- Web dashboard
- Notification integrations (Slack, Discord, Email)
- Advanced rollback strategies
- Cost optimization features
- Plugin system for extensibility

---

## 🙏 Acknowledgments

Built with:

- [Kubebuilder](https://book.kubebuilder.io/) - Operator framework
- [Tekton](https://tekton.dev/) - CI/CD pipelines
- [ArgoCD](https://argo-cd.readthedocs.io/) - GitOps deployment
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controllers

---

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

---

## 💬 Support

- **Documentation**: [docs/](docs/)
- **Team**: Internal project - Contact team members directly

---

Made with ❤️ by Helios Team @ HCMUS
