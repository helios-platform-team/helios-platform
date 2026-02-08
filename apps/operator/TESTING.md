# How to Run Tests

Here are the commands to run tests for the Helios Operator.

## Prerequisites

- Go 1.22+
- Windows (Powershell) or Linux/WSL

## Run All Unit Tests

This runs all unit tests for Controller and GitOps modules.

```powershell
# Run verify all modules
go test -v ./internal/...
```

## Run Specific Tests

**Controller Logic Only:**
```powershell
go test -v ./internal/controller/heliosapp_controller_unit_test.go ./internal/controller/heliosapp_controller.go ./internal/controller/tekton_resources.go ./internal/controller/argocd_resources.go
```

**GitOps Logic Only:**
```powershell
go test -v ./internal/gitops/...
```

## Run with Coverage

```powershell
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```
