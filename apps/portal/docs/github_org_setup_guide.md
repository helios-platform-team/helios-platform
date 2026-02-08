# Backstage GitHub Organization Sync Setup Guide

This guide explains how to configure your Backstage instance to automatically import users and teams from your GitHub Organization.

## 1. Prerequisites

-   A **GitHub Organization**.
-   A **Personal Access Token (PAT)** (Classic) with the following scopes:
    -   `read:org`
    -   `read:user`
    -   `user:email`

## 2. Environment Configuration

You need to provide your GitHub Organization name and your Personal Access Token as environment variables.

In your `.env` file (at the root of `apps/portal`):

```bash
# ... existing vars ...
AUTH_GITHUB_CLIENT_ID=...
AUTH_GITHUB_CLIENT_SECRET=...

# [NEW] Your GitHub Personal Access Token
GITHUB_TOKEN=ghp_your_token_here

# [NEW] Your GitHub Organization Name
GITHUB_ORG=your-org-name
```

## 3. How it Works

### 3.1 App Configuration

We have configured `app-config.yaml` to use the `githubOrg` provider with the correct structure:

```yaml
catalog:
  providers:
    githubOrg:
      id: production
      githubUrl: https://github.com
      orgs: ['${GITHUB_ORG}']
      schedule:
        initialDelay: { seconds: 30 }
        frequency: { minutes: 1 }
        timeout: { minutes: 50 }
```

### 3.2 Backend Registration

And ensuring the correct backend plugin is registered in `packages/backend/src/index.ts`. Note that we use the `github-org` module, NOT the generic `github` module:

```typescript
// GitHub Org Entity Provider
backend.add(import('@backstage/plugin-catalog-backend-module-github-org'));
```

## 4. Verification

1.  Restart your backend: `yarn start`.
2.  Wait a few seconds (approx 30s) for the scheduler to trigger.
3.  Go to the **Catalog** -> **Users** (or **Groups**) in Backstage.
4.  You should see all members of your GitHub Organization automatically listed as Users.
5.  Users can now log in with GitHub and will be matched to these entities.
6.  The authenticated user will see their **GitHub Avatar** in the top-right corner of the Portal (configured in `Root.tsx`).

## 5. Troubleshooting

-   **"User not found"**: If the sync fails, the system falls back to a "Guest" user.
-   **Check Logs**: Look for logs starting with `[Catalog] Read ...` to confirm entities are being ingested.
-   **Validation**: Use the included verify script to check your token permissions:
    ```bash
    cd packages/backend
    node verify_github_sync.js
    ```
