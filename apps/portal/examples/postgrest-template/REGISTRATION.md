# PostgREST Template Registration in Backstage

## Registration Status: ✅ Complete

The PostgREST template has been successfully registered in the Backstage Portal and is now discoverable by users.

## Configuration

**File:** `apps/portal/app-config.yaml`

**Registration Entry:**
```yaml
catalog:
  locations:
    # PostgREST Template - Instant REST API over PostgreSQL
    - type: file
      target: ../../examples/postgrest-template/template.yaml
      rules:
        - allow: [Template]
```

## Template Discovery Path

When Backstage starts, it will:
1. Load the `app-config.yaml` configuration
2. Discover the template at `apps/portal/examples/postgrest-template/template.yaml`
3. Parse the template metadata:
   - **Name:** `postgrest-template`
   - **Title:** PostgREST API Template
   - **Owner:** user:guest
   - **Type:** service
4. Register the template in the catalog
5. Make it available in the **Create** page under available templates

## How Users Access the Template

### Via Backstage UI

1. Open Backstage Portal: `http://localhost:3000`
2. Navigate to **Create** (top menu)
3. Templates are listed alphabetically
4. Find **"PostgREST API Template"** in the list
5. Click to start the scaffolding workflow

### Available Templates (After Registration)

- ✅ Basic Template
- ✅ Advanced Node.js Template  
- ✅ NestJS + Prisma Template
- ✅ **PostgREST API Template** ⭐ (NEW)

## Template Workflow

When users select the PostgREST template, they will:

1. **Step 1: Component Information**
   - Enter service name, port, Docker registry details

2. **Step 2: PostgREST Configuration**
   - Configure API schema (e.g., `public`, `api`)
   - Set JWT secret and role
   - Configure anonymous access role

3. **Step 3: Database Configuration**
   - Choose database settings (auto database trait)

4. **Step 4: Repository & Webhook**
   - Select source repository location

5. **Step 5: Optional Extras**
   - Register in catalog (optional)
   - Send notification (optional)

6. **Steps Execute Automatically**
   - Fetch and publish source code
   - Create webhook for CI/CD
   - Fetch and publish GitOps manifests
   - Create Git credentials secret
   - Deploy HeliosApp to Kubernetes
   - Register component in catalog (if enabled)

## Integration Points

### 1. Backstage Catalog
- Component metadata: `source/catalog-info.yaml`
- Automatically creates API entity for REST endpoints
- Links to database dependency

### 2. Helios Operator
- HeliosApp CRD triggers operator reconciliation
- Operator provisions PostgreSQL database
- Operator injects `PGRST_DB_URI` environment variable

### 3. Tekton CI/CD
- EventListener triggers on source code commits
- PipelineRun builds and pushes container image
- TriggerBinding extracts git events

### 4. ArgoCD GitOps
- Syncs Kubernetes manifests from GitOps repository
- Auto-updates deployments on manifest changes

### 5. Gitea
- Stores source code repositories
- Stores GitOps manifests
- Manages webhook credentials

## Verification

To verify the template is properly registered:

1. **Check Configuration File:**
   ```bash
   grep -A 3 "postgrest-template" apps/portal/app-config.yaml
   ```

2. **Validate Template YAML:**
   ```bash
   cd apps/portal/examples/postgrest-template
   python3 << 'EOF'
   import yaml
   with open('template.yaml', 'r') as f:
       template = yaml.safe_load(f)
       print(f"Template Name: {template['metadata']['name']}")
       print(f"Template Title: {template['metadata']['title']}")
       print(f"Spec Type: {template['spec']['type']}")
       print(f"Parameters: {len(template['spec']['parameters'])} sections")
       print(f"Steps: {len(template['spec']['steps'])} execution steps")
   EOF
   ```

3. **Run Backstage UI Tests (after deployment):**
   ```bash
   # Navigate to Backstage Portal
   # Check Create page to see PostgREST template appears
   # Click template to verify scaffolding UI renders
   ```

## Next Steps

1. **Start Backstage Portal:**
   ```bash
   cd apps/portal
   npm install
   npm run start
   ```

2. **Navigate to Create Page:**
   - Open http://localhost:3000
   - Click **Create** in top menu
   - Verify PostgREST template appears

3. **Test Template Scaffolding:**
   - Fill in template parameters
   - Execute scaffolding workflow
   - Verify resources created in Gitea
   - Check Helios Operator reconciliation
   - Verify database provisioned
   - Test PostgREST API connectivity

4. **Deployment to Production:**
   - Build Docker image for Backstage Portal
   - Deploy to Kubernetes cluster
   - Configure persistent catalog storage (e.g., PostgreSQL)
   - Enable GitHub/GitLab integration for applica discovery

## Troubleshooting

**Template not appearing in Backstage UI:**
1. Check app-config.yaml for syntax errors: `yaml -c app-config.yaml`
2. Verify template file exists: `ls -la apps/portal/examples/postgrest-template/template.yaml`
3. Check Backstage backend logs for catalog loading errors
4. Restart Backstage backend if changes made to app-config.yaml

**Template parameters not rendering:**
1. Verify Handlebars syntax in template files: `${{ values.name }}`
2. Check that all referenced parameters are defined in parameters section of template.yaml
3. Review Backstage template documentation at https://backstage.io/docs/features/software-templates/

**HeliosApp not deploying:**
1. Verify Helios Operator is running: `kubectl get pods -n helios`
2. Check operator logs: `kubectl logs -n helios -l app=operator -f`
3. Verify HeliosApp CRD is registered: `kubectl get crd | grep helios`
4. Check database trait configuration in template

## Files Modified

- `apps/portal/app-config.yaml` — Added PostgREST template to catalog locations

## Files Created (Previously)

- `apps/portal/examples/postgrest-template/template.yaml`
- `apps/portal/examples/postgrest-template/content/source/` (5 files)
- `apps/portal/examples/postgrest-template/content/gitops/` (5 files)

---

**Status:** ✅ Ready for testing and deployment
**Last Updated:** April 6, 2026
