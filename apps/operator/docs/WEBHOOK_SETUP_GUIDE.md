# GitHub Webhook Setup Guide

Note: For a quick, streamlined setup, follow the simplified instructions in GETTING_STARTED.md (see Step 6: "Setup GitHub Webhooks"). This page is the detailed reference with extra options and troubleshooting.

Quick reference for setting up GitHub webhooks to auto-trigger Helios pipelines.

## Prerequisites

- ✅ Helios Operator running
- ✅ Tekton EventListener deployed
- ✅ Webhook secret created (`github-webhook-secret`)
- ✅ EventListener service exposed (publicly accessible)

## Quick Setup Steps

### 1. Get Your Webhook URL

Choose one method to expose your EventListener:

**For Development (ngrok):**

```bash
kubectl port-forward -n default svc/el-helios-listener 8080:8080 &
ngrok http 8080
```

Copy the HTTPS URL: `https://abc123.ngrok.io`

**For Production (Ingress):**

Your webhook URL will be: `https://webhook.yourdomain.com`

### 2. Get Your Webhook Secret

```bash
kubectl get secret github-webhook-secret -o jsonpath='{.data.secretToken}' | base64 -d
echo ""
```

Copy this secret value.

### 3. Configure GitHub Webhook

Go to: `https://github.com/YOUR_USERNAME/YOUR_REPO/settings/hooks`

Click **Add webhook** and fill in:

| Field                | Value                                                                 |
| -------------------- | --------------------------------------------------------------------- |
| **Payload URL**      | `https://YOUR_URL/hooks`                                              |
| **Content type**     | `application/json`                                                    |
| **Secret**           | `[Paste your webhook secret]`                                         |
| **SSL verification** | Enable (for production with valid cert) / Disable (for ngrok/testing) |
| **Events**           | Select "Just the push event"                                          |
| **Active**           | ✅ Checked                                                            |

Click **Add webhook**.

## GitHub Webhook Form - Field Details

### Payload URL

**Format:** `https://YOUR_EXPOSED_URL/hooks`

**Examples:**

- ngrok: `https://abc123.ngrok.io/hooks`
- LoadBalancer: `http://34.123.45.67:8080/hooks`
- Ingress: `https://webhook.example.com/hooks`

**Important:**

- Must be publicly accessible from the internet
- Must end with `/hooks`
- Use HTTPS in production

### Content type

**Select:** `application/json`

This tells GitHub to send the webhook payload as JSON format.

### Secret

**Value:** Your webhook secret token

Get it with:

```bash
kubectl get secret github-webhook-secret -o jsonpath='{.data.secretToken}' | base64 -d
```

This secret is used to verify that webhook requests are from GitHub and haven't been tampered with.

### SSL verification

**Enable SSL verification:** ✅ Recommended for production

- Requires valid SSL certificate
- More secure

**Disable SSL verification:** ⚠️ Only for testing

- Use when testing with ngrok
- Use when using self-signed certificates
- **NOT recommended for production**

### Which events would you like to trigger this webhook?

**Option 1: Just the push event** (Recommended for most cases)

- ✅ Select this radio button
- Triggers webhook only when code is pushed to any branch

**Option 2: Let me select individual events**
Select this if you want more control:

- ✅ **Pushes** - Trigger on push to repository
- ✅ **Pull requests** - Trigger on PR events (optional)
- ✅ **Releases** - Trigger on release creation (optional)

#### Option 3: Send me everything

- ⚠️ Not recommended - too many events

### Active

**Always check this box:** ✅

This enables the webhook. Unchecked webhooks won't send events.

## Test Your Webhook

### 1. Make a Test Push

```bash
cd ~/my-nodejs-app
echo "// Test webhook" >> index.js
git add index.js
git commit -m "Test webhook trigger"
git push origin main
```

### 2. Check GitHub Delivery

1. Go to your webhook settings
2. Click on the webhook
3. Scroll to **Recent Deliveries**
4. Look for green checkmark (✅) = Success

### 3. Verify Pipeline Triggered

```bash
# Should see new PipelineRun
kubectl get pipelinerun -l heliosapp=my-nodejs-app --sort-by=.metadata.creationTimestamp

# Check EventListener logs
kubectl logs -l eventlistener=helios-listener --tail=50
```

## Troubleshooting

### ❌ Webhook Failed: Connection Refused

**Cause:** GitHub cannot reach your EventListener URL

**Fix:**

```bash
# Verify service is running
kubectl get svc el-helios-listener

# Check pods are ready
kubectl get pods -l eventlistener=helios-listener

# Test locally first
curl -X POST http://localhost:8080/hooks \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{}'

# For ngrok - make sure tunnel is active
ngrok http 8080
```

### ❌ Webhook Failed: 401 Unauthorized

**Cause:** Webhook secret mismatch

**Fix:**

```bash
# Verify secret value
kubectl get secret github-webhook-secret -o jsonpath='{.data.secretToken}' | base64 -d

# Update GitHub webhook with correct secret
# Go to GitHub > Settings > Webhooks > Edit > Update Secret field
```

### ❌ Webhook Success but No PipelineRun

**Cause:** EventListener not configured correctly or TriggerBinding/TriggerTemplate missing

**Fix:**

```bash
# Check EventListener logs for errors
kubectl logs -l eventlistener=helios-listener --tail=100

# Verify resources exist
kubectl get eventlistener,triggerbinding,triggertemplate

# Check Tekton Triggers are working
kubectl get pods -n tekton-pipelines | grep trigger
```

### ❌ SSL Verification Failed

**Cause:** Invalid or self-signed SSL certificate

**Temporary Fix (Development only):**

- In GitHub webhook settings, uncheck "Enable SSL verification"

**Permanent Fix (Production):**

- Use Let's Encrypt with cert-manager
- Use a valid SSL certificate from a trusted CA
- Configure Ingress with proper TLS

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer for Let's Encrypt
cat > letsencrypt-issuer.yaml << EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

kubectl apply -f letsencrypt-issuer.yaml
```

## Webhook Payload Example

When GitHub sends a webhook, it looks like this:

```json
{
  "ref": "refs/heads/main",
  "before": "abc123...",
  "after": "def456...",
  "repository": {
    "name": "my-nodejs-app",
    "full_name": "YOUR_USERNAME/my-nodejs-app",
    "clone_url": "https://github.com/YOUR_USERNAME/my-nodejs-app.git",
    "ssh_url": "git@github.com:YOUR_USERNAME/my-nodejs-app.git"
  },
  "pusher": {
    "name": "YOUR_USERNAME",
    "email": "you@example.com"
  },
  "commits": [
    {
      "id": "def456...",
      "message": "Test webhook trigger",
      "timestamp": "2025-10-19T10:00:00Z",
      "author": {
        "name": "Your Name",
        "email": "you@example.com"
      }
    }
  ]
}
```

## Headers GitHub Sends

```text
X-GitHub-Event: push
X-GitHub-Delivery: abc-123-def-456
X-Hub-Signature-256: sha256=...
Content-Type: application/json
User-Agent: GitHub-Hookshot/...
```

## Security Best Practices

1. **Always use HTTPS in production**

   - Protects webhook payload from eavesdropping
   - Prevents man-in-the-middle attacks

2. **Always set a webhook secret**

   - Verifies requests are from GitHub
   - Prevents replay attacks

3. **Enable SSL verification**

   - Only disable for local testing
   - Use valid certificates in production

4. **Restrict webhook events**

   - Only enable events you need
   - Reduces unnecessary processing

5. **Monitor webhook deliveries**

   - Check Recent Deliveries regularly
   - Set up alerts for failed webhooks

6. **Use IP allowlisting (optional)**

- GitHub webhook IPs: <https://api.github.com/meta>
- Add firewall rules to only allow GitHub IPs

## Exposing EventListener - Detailed Options

### Option 1: ngrok (Development/Testing)

**Pros:**

- ✅ Quick and easy
- ✅ Works with local clusters
- ✅ HTTPS automatically

**Cons:**

- ❌ URL changes on restart
- ❌ Not for production
- ❌ May have rate limits

**Setup:**

```bash
# Download ngrok
# https://ngrok.com/download

# Expose service
kubectl port-forward svc/el-helios-listener 8080:8080 &
ngrok http 8080

# Use the HTTPS URL provided
```

### Option 2: LoadBalancer (Cloud Clusters)

**Pros:**

- ✅ Persistent IP
- ✅ Good for cloud environments
- ✅ Production-ready

**Cons:**

- ❌ Costs money (cloud provider charges)
- ❌ Requires cloud cluster
- ❌ Need to setup SSL separately

**Setup:**

```bash
kubectl patch svc el-helios-listener -p '{"spec": {"type": "LoadBalancer"}}'
kubectl get svc el-helios-listener
# Use the EXTERNAL-IP
```

### Option 3: Ingress (Recommended for Production)

**Pros:**

- ✅ Production-ready
- ✅ Can use custom domain
- ✅ SSL with cert-manager
- ✅ Multiple services on one IP

**Cons:**

- ❌ Requires Ingress controller
- ❌ More complex setup
- ❌ Need DNS configuration

**Setup:**

```bash
# Install nginx ingress controller (if not present)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml

# Create ingress
kubectl apply -f webhook-ingress.yaml
```

## Complete Workflow with Webhooks

```text
Developer → git push
    ↓
GitHub → Webhook POST to EventListener
    ↓
EventListener → Validates secret
    ↓
TriggerBinding → Extracts data from webhook
    ↓
TriggerTemplate → Creates PipelineRun
    ↓
Tekton Pipeline → Builds & Deploys
    ↓
ArgoCD → Syncs to cluster
    ↓
Application Updated! 🚀
```

## Additional Resources

- [GitHub Webhooks Documentation](https://docs.github.com/en/webhooks)
- [Tekton Triggers Documentation](https://tekton.dev/docs/triggers/)
- [ngrok Documentation](https://ngrok.com/docs)
- [cert-manager Documentation](https://cert-manager.io/docs/)

## Quick Commands Reference

```bash
# Get webhook secret
kubectl get secret github-webhook-secret -o jsonpath='{.data.secretToken}' | base64 -d

# Check EventListener
kubectl get eventlistener
kubectl logs -l eventlistener=helios-listener -f

# Test webhook locally
curl -X POST http://localhost:8080/hooks \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=test" \
  -d '{"ref":"refs/heads/main","repository":{"clone_url":"https://github.com/user/repo"}}'

# Watch for new PipelineRuns
kubectl get pipelinerun -w

# Expose with port-forward
kubectl port-forward svc/el-helios-listener 8080:8080

# Check recent events
kubectl get events --sort-by='.lastTimestamp' | tail -20
```

---

**Need help?** Check the main [GETTING_STARTED.md](./GETTING_STARTED.md) guide or the troubleshooting section above.
