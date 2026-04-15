# PR #68 Validation Evidence Guide

This checklist is prepared for PR https://github.com/helios-platform-team/helios-platform/pull/68.

Goal: provide proof that the .NET template can be created, CI/CD runs, and deployment succeeds.

## Versions Used

- .NET SDK: 10.0.202
- .NET runtime: 10.0.5
- PostgreSQL: 18.3

## Required Screenshots

Save screenshots under a local folder like `docs/evidence/pr68/` and attach them to the PR comment.

1. Template creation form completed in Backstage
- Page: Scaffolder template form for .NET template
- Show fields: service name, port, repo, db settings (PostgreSQL 18.3)
- Suggested filename: `01-template-form.png`

2. Scaffolder run success
- Page: Task/Job result page after clicking Create
- Show green/completed status and generated source + gitops repo links
- Suggested filename: `02-scaffolder-success.png`

3. Generated HeliosApp manifest includes database version 18.3
- Page: Generated gitops repo file `helios-app.yaml`
- Show database trait block with version `18.3`
- Suggested filename: `03-heliosapp-db-version.png`

4. Tekton pipeline run succeeded
- Page: Kubernetes/Tekton view (PipelineRuns)
- Show latest run status `Succeeded`
- Suggested filename: `04-tekton-pipelinerun-succeeded.png`

5. ArgoCD application synced and healthy
- Page: ArgoCD application detail
- Show `Synced` and `Healthy`
- Suggested filename: `05-argocd-synced-healthy.png`

6. Running service endpoint proof
- Page: app endpoint or swagger page
- Show live response (for example `/health` => status ok)
- Suggested filename: `06-service-health.png`

## Optional CLI Evidence (copy into PR comment)

```bash
# Show app + DB trait in generated gitops manifest
rg -n "database|version|dbType|dbName" path/to/generated-gitops/helios-app.yaml -n

# Show Tekton run status
kubectl get pipelineruns -n default

# Show deployed resources for generated app
kubectl get deploy,svc,sts,pods -n default | rg "<your-app-name>|postgres"
```

## Suggested PR Comment Template

```md
Validation evidence for PR #68 (.NET template):

- .NET template creation: PASS
- GitOps repo generated: PASS
- Database trait uses PostgreSQL 18.3: PASS
- Tekton PipelineRun: SUCCEEDED
- ArgoCD sync: SYNCED + HEALTHY
- Service endpoint: reachable (`/health` returns ok)

Screenshots attached:
1. 01-template-form.png
2. 02-scaffolder-success.png
3. 03-heliosapp-db-version.png
4. 04-tekton-pipelinerun-succeeded.png
5. 05-argocd-synced-healthy.png
6. 06-service-health.png
```
