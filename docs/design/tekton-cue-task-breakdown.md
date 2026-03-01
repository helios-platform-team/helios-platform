# Task Breakdown: Tekton to CUE Migration

> **Issue:** [Analysis] Map Tekton Resources to CUE Schema #18  
> **Date:** 2026-02-04  
> **Review Policy:** All 5 members review every task before merge

---

## Team & Workload Balance

| Member | Team | Other Duties | This Task |
|--------|------|--------------|-----------|
| Phước Hoàn | Team 1 - Infra | None | **12h** |
| Kha | Team 3 - Fullstack | Backstage tasks | 6h |
| Nghĩa | Team 3 - Fullstack | Backstage tasks | 6h |
| Việt Hoàng | Team 3 - Fullstack | Backstage tasks | 6h |
| Ngọc Anh | Team 3 - Nhóm trưởng | Backstage tasks | 6h |

---

## Task Overview

| Task | Owner | Effort | Phase |
|------|-------|--------|-------|
| T1: CUE Foundation & Bases | Phước Hoàn | 6h | Phase 1 |
| T2: CUE Task Definitions & Registry | Kha | 6h | Phase 1 |
| T3: CUE Pipeline, Pattern & Registry | Nghĩa | 6h | Phase 1 |
| T4: CUE Triggers & Registry | Việt Hoàng | 6h | Phase 1 |
| T5: Go SDK & Operator Refactor | Phước Hoàn | 6h | Phase 1 |
| T6: E2E Testing & Cleanup | Ngọc Anh | 6h | Phase 2 |

**Total: 36h**

---

## T1: CUE Foundation & Bases

| Attribute | Value |
|-----------|-------|
| **Owner** | Phước Hoàn |
| **Effort** | 6 hours |
| **Phase** | Phase 1, Day 1-2 |

### What to do

1. **Input schema** (`_schema.cue`)
   - `#TektonInput` với all fields từ HeliosApp CRD
   - Thêm `pipelineType` và `triggerType` với default values (`| *`)

2. **Shared definitions** (`_common.cue`)
   - `#CommonParams`: Shared parameters
   - `#Defaults`: Default images, secrets
   - `#Labels`: Common K8S labels

3. **Base templates** (`bases/`)
   - `_base.cue`: Common K8S resource structure
   - `task.cue`: `#TektonTask` template
   - `pipeline.cue`: `#TektonPipeline` template
   - `trigger.cue`: `#TriggerBundle` và base trigger types

### Acceptance Criteria

| # | Criteria | Verify |
|---|----------|--------|
| 1 | CUE compiles | `cue vet ./definitions/tekton/...` |
| 2 | `#TektonPipeline` correctly generates Pipeline spec | `cue export` base template |
| 3 | DRY principle | No duplicate definitions across base files |
| 4 | Bases support extension | Verify using an example extension |

---

## T2: CUE Task Definitions & Registry

| Attribute | Value |
|-----------|-------|
| **Owner** | Kha |
| **Effort** | 6 hours |
| **Phase** | Phase 1, Day 3-4 |
| **Depends** | T1 |

### What to do

1. **Task registry** (`tasks/_registry.cue`)
   - Implement `#TaskRegistry` map
   - Implement `#RenderTask` helper

2. **git-clone task** (`tasks/git-clone.cue`)
   - Use `#TektonTask` base
   - Match current implementation logic

3. **kaniko-build task** (`tasks/kaniko-build.cue`)
   - Docker config volume management
   - Results definition

4. **git-update-manifest task** (`tasks/git-update.cue`)
   - GitOps update logic

### Acceptance Criteria

| # | Criteria | Verify |
|---|----------|--------|
| 1 | Tasks compile | `cue vet ./definitions/tekton/tasks/...` |
| 2 | `#RenderTask` returns correct task from registry | `cue export` with `taskType` |
| 3 | Uses `#CommonParams` | Review code - no hardcoded params in tasks |
| 4 | Registry pattern follows Open/Closed | Add new task without modifying bases |

---

## T3: CUE Pipeline, Pattern & Registry

| Attribute | Value |
|-----------|-------|
| **Owner** | Nghĩa |
| **Effort** | 6 hours |
| **Phase** | Phase 1, Day 3-4 |
| **Depends** | T1 |

### What to do

1. **Reusable patterns** (`pipelines/_patterns.cue`)
   - `#FetchSourcePattern`, `#BuildImagePattern`, `#UpdateGitOpsPattern`
   - `#StandardPipelineParams`, `#StandardWorkspaces`

2. **Pipeline registry** (`pipelines/_registry.cue`)
   - Implement `#PipelineRegistry`
   - Implement `#RenderPipeline` helper

3. **from-code-to-cluster pipeline** (`pipelines/from-code-to-cluster.cue`)
   - Use `#TektonPipeline` base
   - Compose from patterns

### Acceptance Criteria

| # | Criteria | Verify |
|---|----------|--------|
| 1 | Pipeline compiles | `cue vet ./definitions/tekton/pipelines/...` |
| 2 | Output matches `apps/operator/tekton/pipeline.yaml` | `cue export` and diff |
| 3 | Registry allows multiple pipelines | Add dummy pipeline to registry |
| 4 | Patterns are truly reusable | Compose a new pipeline using existing patterns |

---

## T4: CUE Triggers & Registry

| Attribute | Value |
|-----------|-------|
| **Owner** | Việt Hoàng |
| **Effort** | 6 hours |
| **Phase** | Phase 1, Day 3-5 |
| **Depends** | T1, T3 |

### What to do

1. **Trigger registry** (`triggers/_registry.cue`)
   - Implement `#TriggerRegistry`
   - Implement `#RenderTriggers` helper

2. **GitHub push trigger bundle** (`triggers/github-push.cue`)
   - `#GitHubPushTriggerBinding`, `#BuildTriggerTemplate`, `#GitHubEventListener`
   - Wrap in `#GitHubPushTriggerBundle`

3. **Webhook ingress** (`triggers/webhook-ingress.cue`)

4. **Finalize builder** (`engine/tekton_builder.cue`)
   - Wire all registries (Task, Pipeline, Trigger)
   - Use `tektonInput.pipelineType` and `tektonInput.triggerType` with defaults

### Acceptance Criteria

| # | Criteria | Verify |
|---|----------|--------|
| 1 | Builder renders all objects | `cue export ./engine/tekton_builder.cue` |
| 2 | Triggers are correctly grouped in bundles | Review registry structure |
| 3 | Builder uses Registry Pattern (No hardcoding) | Review builder logic |
| 4 | Conditional rendering works | Verify output with/without webhookDomain |

---

## T5: Go SDK & Operator Refactor

| Attribute | Value |
|-----------|-------|
| **Owner** | Phước Hoàn |
| **Effort** | 6 hours |
| **Phase** | Phase 1, Day 5 → Week 2 Day 1-2 |
| **Depends** | T2, T3, T4 |

### What to do

1. **Go SDK Integration**
   - Add `cuelang.org/go/cue` dependency
   - Embed `.cue` files in binary

2. **CUE Engine Wrapper** (`internal/cue/tekton.go`)
   - Implement `RenderTektonResources(input TektonInput)`
   - Type mapping between Go and CUE

3. **Controller Refactor** (`heliosapp_controller.go`)
   - Replace old `Generate*` hardcoded calls
   - Add feature flag `USE_CUE_TEKTON` for safe rollout

4. **Unit testing** for the new renderer

### Acceptance Criteria

| # | Criteria | Verify |
|---|----------|--------|
| 1 | Operator builds successfully | `go build ./...` |
| 2 | Unit tests cover all resource types | `go test` output |
| 3 | Rollback mechanism works | Toggle feature flag and verify logic |
| 4 | Renderer handles mapping errors gracefully | Test with invalid input |

---

## T6: E2E Testing & Cleanup

| Attribute | Value |
|-----------|-------|
| **Owner** | Ngọc Anh |
| **Effort** | 6 hours |
| **Phase** | Phase 2 |
| **Depends** | T5 |

### What to do

1. **E2E Validation (4h)**
   - Deploy Operator with CUE enabled
   - Verify full CI/CD flow from Webhook to Deployment
   - Compare YAML outputs in cluster with old version

2. **Cleanup & Deprecation (2h)**
   - Mark old Go functions as `// Deprecated`
   - Remove feature flag after successful validation period
   - Delete legacy code in `tekton_resources.go`
   - Finalize architecture documentation

### Acceptance Criteria

| # | Criteria | Verify |
|---|----------|--------|
| 1 | Successful PipelineRun using CUE-generated resources | Check Tekton dashboard/logs |
| 2 | Identical resource structure to Go version | `kubectl get -o yaml` diff |
| 3 | Old code safely removed | Review codebase for legacy functions |
| 4 | Documentation represents the final architecture | Review `docs/` |

---

## Master Plan CSV

| Task | Priority | Owner | Phase |
|------|----------|-------|-------|
| [Infra] CUE Foundation & Bases | P0 | Phước Hoàn | Phase 1 |
| [Infra] CUE Task Definitions & Registry | P0 | Kha | Phase 1 |
| [Infra] CUE Pipeline & Registry | P0 | Nghĩa | Phase 1 |
| [Infra] CUE Triggers & Registry | P1 | Việt Hoàng | Phase 1 |
| [Operator] Go SDK & Refactor | P0 | Phước Hoàn | Phase 1 |
| [Testing] E2E & Cleanup | P0 | Ngọc Anh | Phase 2 |
