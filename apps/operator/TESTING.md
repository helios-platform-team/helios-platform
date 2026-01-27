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
# option 1: Run all tests in controller package
go test -v ./internal/controller/...

# option 2: Run specific test file (requires including implementation files or just testing the package)
go test -v ./internal/controller/
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
