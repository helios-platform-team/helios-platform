# Templates

This folder contains example/template YAMLs that are not applied directly to your cluster. They are reference files:

- `tekton-pipeline-template.yaml` – example of a minimal Tekton Pipeline
- `tekton-trigger-template.yaml` – example TriggerTemplate + EventListener
- `application-template.yaml` – example ArgoCD Application spec

Use them as guides if you want to customize resources. The actual, runnable resources live under `tekton/` and are wired to the operator.
