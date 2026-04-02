---
name: implement-task
description: Implementation workflow for the Helios Platform operator. Use when implementing features, fixing bugs, or making any code changes. Enforces best practices, testing, and verification.
---

# Implementation Task Workflow

## Detected Stack

!`grep -rh "^go " --include="go.mod" . 2>/dev/null | head -1 | grep . && grep -rh 'language.*version' --include="module.cue" cue.mod/ cue/cue.mod/ 2>/dev/null | head -1 | grep . || echo "unknown stack"`

## Workflow

When asked to implement a task, follow this exact sequence. Do NOT skip steps.

---

### 1. Understand & Plan

- Read and understand the full scope of the task
- Identify all files that need to change
- Check existing code patterns in the codebase — follow them
- If the task is non-trivial (>3 files), create a brief plan and get user approval before coding

### 2. Implement with Best Practices

**Code quality principles — apply ALL of these:**

- **SOLID** — Single responsibility, open/closed, Liskov substitution, interface segregation, dependency inversion
- **DRY** — Don't repeat yourself. Extract reusable functions, constants, and definitions
- **KISS** — Keep it simple. Prefer clarity over cleverness
- **YAGNI** — Don't build what isn't needed yet
- **Separation of Concerns** — Each module/file/function has one clear purpose
- **Defensive Programming** — Validate inputs, handle edge cases, never swallow errors
- **Fail Fast** — Return errors immediately, don't accumulate them silently

**Go-specific (use `use-modern-go` skill):**

- Use modern Go idioms: `any`, `cmp.Or`, `slices`, `maps`, `errors.Is/As`
- Proper error wrapping with `fmt.Errorf("context: %w", err)`
- Table-driven tests with descriptive names
- Context propagation for cancellation and timeouts
- Meaningful variable/function names — no single-letter names outside loops
- Comments on exported types and functions (godoc convention)

**CUE-specific (use `use-modern-cue` skill):**

- Use `#` definitions for schemas, `_` for hidden fields, `let` for intermediates
- Validate with constraints, not imperative logic
- Follow the module's established patterns (registries, builders, etc.)
- File-level `@extern(embed)` when using `@embed`

**Non-functional requirements:**

- **Performance** — avoid unnecessary allocations, use efficient algorithms
- **Reliability** — handle all error paths, use retries where appropriate
- **Observability** — add structured logging at key decision points
- **Security** — no hardcoded secrets, validate all external input

### 3. Test Everything

After implementation, run ALL of these. Do NOT stop at the first passing test.

```sh
# From apps/operator/ directory:

# 1. Compilation check — must pass cleanly
go build ./...

# 2. Static analysis
go vet ./...

# 3. Full test suite with envtest (real K8s API server)
make test

# 4. Build the operator binary
make build

# 5. Verify CUE evaluation (if CUE files changed)
# Run CUE-related tests which validate CUE rendering
go test ./internal/cue/... -v -count=1
```

**If any test fails:**
1. Read the error output carefully
2. Fix the root cause (not symptoms)
3. Re-run ALL tests, not just the one that failed
4. If a fix is complex or risky, explain the issue and ask the user before proceeding

**If a dependency/infrastructure issue prevents testing:**
1. Document what couldn't be tested and why
2. Explain the risk to the user
3. Suggest how to test manually

### 4. Verify & Report

After all tests pass, verify:

- [ ] `go build ./...` — compiles cleanly
- [ ] `go vet ./...` — no issues
- [ ] `make test` — all tests pass with envtest
- [ ] `make build` — operator binary builds
- [ ] No warnings in any output
- [ ] CRD manifests regenerated if API types changed (`controller-gen` runs in `make build`)

**Report to user:**
- What was implemented (brief)
- What was tested (specific results)
- Any caveats or follow-up items

---

## Rollback Strategy

If something doesn't work after implementation:

1. Identify the specific change that broke things
2. Revert that change only (not everything)
3. Re-run all tests to confirm the rollback works
4. Explain what went wrong and propose an alternative approach

---

## Project-Specific Context

**Directory structure:**
- `apps/operator/` — Go operator (controllers, CUE engine, GitOps)
- `cue/` — CUE definitions (schemas, tekton, components, traits, engine)
- `apps/operator/internal/cue/` — Go↔CUE bridge (engine.go, tekton.go)
- `apps/operator/internal/controller/` — Kubernetes controllers
- `apps/operator/internal/gitops/` — Git operations

**Key patterns:**
- CUE handles resource generation (Tasks, Pipeline, Triggers, Ingress)
- Go handles runtime resources (PipelineRun, RBAC, ArgoCD Application)
- Controller reconciles HeliosApp CRD → CUE rendering → GitOps sync → ArgoCD
- Tests use envtest (real K8s API) + CUE engine unit tests + E2E validation tests
