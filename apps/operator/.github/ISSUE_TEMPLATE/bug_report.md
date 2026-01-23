---
name: Bug Report
about: Create a report to help us improve
title: "[BUG] "
labels: ["bug", "triage"]
assignees: ""
---

## Bug Description

A clear and concise description of what the bug is.

## Steps To Reproduce

1. Deploy operator with '...'
2. Create HeliosApp CR '...'
3. Check logs '...'
4. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

What actually happened.

## Environment

- **Operator Version**: [e.g., v1.0.0]
- **Kubernetes Version**: [e.g., v1.34.1]
- **Platform**: [e.g., Kind, GKE, EKS, AKS]
- **Go Version**: [e.g., 1.25.3]

## Logs

```
Paste relevant logs here
```

## Configuration

```yaml
# Paste your HeliosApp CR here
apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: example
spec: ...
```

## Additional Context

Add any other context about the problem here (screenshots, related issues, etc.)

## Possible Solution

If you have ideas on how to fix this, please share.
