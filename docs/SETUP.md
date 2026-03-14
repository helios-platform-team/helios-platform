# Setup Guide

This guide explains how to install necessary development tools for the Helios Platform.

> **Tip:** For a fully automated setup, see the [Quick Start](APP_STARTUP_GUIDE.md#quick-start-recommended) in the App Startup Guide. Run `task check` to verify your tooling at any time.

## 1. Prerequisites

Ensure you have the following installed:

- **Task**: [taskfile.dev](https://taskfile.dev/) (the project task runner)
- **Go**: v1.24.0+
- **Docker**: 17.03+
- **kubectl**: v1.11.3+
- **Node.js**: v22+ (for Backstage portal)
- **Yarn**: v4+ (`corepack enable && corepack prepare yarn@4 --activate`)

## 2. Install CLI Tools

Install the essential CLIs for template management and local cluster testing:

```bash
# Install CUE
go install cuelang.org/go/cmd/cue@latest

# Install k3d (lightweight k3s-in-Docker)
# Linux / macOS
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Windows (Scoop)
scoop install k3d

# Windows (Chocolatey)
choco install k3d
```

## 3. Configure PATH

Ensure the Go binary directory is in your shell's `PATH` to run the tools globally:

```bash
# Linux / macOS: add to ~/.zshrc or ~/.bashrc
export PATH="$PATH:$(go env GOPATH)/bin"
```

On **Windows**, add `%GOPATH%\bin` to your system PATH via System Environment Variables, or run:

```powershell
[System.Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";" + (go env GOPATH) + "\bin", "User")
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
