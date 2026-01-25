
## GitOps Configuration

To use the GitOps features, you must configure a Kubernetes Secret with your GitHub credentials.

### 1. Create the Secret
The operator expects a secret (default name: `github-webhook-secret`) containing your GitHub Personal Access Token (PAT) and username.

**Required Keys:**
- `username`: Your GitHub username (or the owner of the token).
- `token`: Your Classic PAT with `repo` scope.

```bash
kubectl create secret generic github-webhook-secret \
  --from-literal=username=YOUR_GITHUB_USERNAME \
  --from-literal=token=YOUR_CLASSIC_TOKEN_HERE \
  --namespace=default
```

### 2. Troubleshooting

**Error: `authentication required: Invalid username or token`**
- **Cause:** The token is invalid, expired, or the username does not match the token owner. It can also happen if the `username` key is missing from the secret (defaulting to "git").
- **Fix:** Recreate the secret ensuring both `username` and `token` are correct. Verify the token scope includes `repo`.

**Error: Secret Updates Not Taking Effect**
- **Cause:** The Operator might be caching the old secret or running older code if verified locally via `make run`.
- **Fix:** If running locally with `make run`, you MUST stop (Ctrl+C) and restart the process to pick up code changes or environment updates. For cluster deployments, restart the operator pod.

**Error: `Secret "..." not found` despite being created**
- **Cause:** The `HeliosApp` CR in the cluster might still be referencing an old secret name (or a raw token value if wrongly configured previously).
- **Fix:** Ensure your YAML manifest references the *name* of the secret:
  ```yaml
  gitopsSecretRef: "github-webhook-secret"
  ```
  Then re-apply the manifest: `kubectl apply -f config/samples/app_v1alpha1_heliosapp.yaml`.
