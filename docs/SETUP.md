# Setup Guide

This guide explains how to install necessary development tools for the Helios Platform.

## 1. Prerequisites

Ensure you have the following installed:

- **Go**: v1.24.0+
- **Docker**: 17.03+
- **kubectl**: v1.11.3+

## 2. Install CLI Tools

Install the essential CLIs for template management and local cluster testing:

```bash
# Install CUE
go install cuelang.org/go/cmd/cue@latest

# Install Kind
go install sigs.k8s.io/kind@latest
```

## 3. Configure PATH

Ensure the Go binary directory is in your shell's `PATH` to run the tools globally:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$PATH:$(go env GOPATH)/bin"
```

## 4. Install Operator Development Tools

Download project-specific versions of Kustomize, Operator SDK, and Test Binaries:

```bash
make -C apps/operator kustomize controller-gen operator-sdk setup-envtest
```

## 4. Verify Setup

Run the CUE engine integration tests to ensure everything is configured correctly:

```bash
go -C apps/operator test -v ./internal/cue/...
```
