# Helios Database Backend Plugin

This plugin provides a backend service for fetching database connectivity information from Kubernetes secrets and pod status.

## Features

- Fetches database connectivity details from Kubernetes secrets
- Retrieves database pod status from Kubernetes
- Provides REST API endpoint for frontend consumption
- Secure password handling with base64 encoding/decoding

## API Endpoint

### GET `/api/helios/database/info/:componentName`

Fetches database information for a specified component.

**Parameters:**
- `componentName` (required): The name of the component (e.g., "test-db-app-v17")

**Response:**
```json
{
  "host": "localhost",
  "port": 5432,
  "user": "postgres",
  "password": "secretpassword",
  "database": "mydb",
  "status": "Running"
}
```

**Expected Kubernetes Resources:**

The plugin expects the following Kubernetes resources:

1. **Secret**: Named `{componentName}-backend-db-secret`
   - Should contain:
     - `host`: Database host
     - `port`: Database port
     - `user`: Database user
     - `password`: Database password (base64 encoded)
     - `database`: Database name

2. **Pod**: Labeled with `app={componentName}-backend-db`
   - Used for retrieving the database pod status

**Example Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-db-app-v17-backend-db-secret
  namespace: default
type: Opaque
data:
  host: bG9jYWxob3N0  # localhost
  port: NTQzMg==  # 5432
  user: cG9zdGdyZXM=  # postgres
  password: c2VjcmV0cGFzc3dvcmQ=  # secretpassword
  database: bXlkYg==  # mydb
```

## Installation

1. The plugin is automatically included in the workspace via the `plugins/*` configuration
2. The backend service is registered in `packages/backend/src/index.ts`
3. Dependencies are managed through yarn workspaces

## Security Considerations

- The plugin directly accesses Kubernetes secrets
- Requires a ServiceAccount with permissions to read secrets and list pods
- Password data is base64-encoded in transit but should be handled securely
- In production, consider implementing additional authentication and encryption

## Example ServiceAccount Configuration

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backstage-sa
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: backstage-database-reader
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: backstage-database-reader-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: backstage-database-reader
subjects:
  - kind: ServiceAccount
    name: backstage-sa
    namespace: default
```

## Development

### Building the plugin

```bash
yarn build:backend
```

### Testing

```bash
yarn test
```
