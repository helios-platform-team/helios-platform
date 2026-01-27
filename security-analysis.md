# Security Analysis: Docker Credentials in Scaffolder Templates

## 1. The Risk: Committing Secrets to Git
The current approach (taking credentials from the form and rendering a `Secret` manifest) results in:
*   **Plaintext/Base64 in Git**: The `secret.yaml` file in the GitOps repository contains the base64-encoded credentials. Anyone with **Read** access to that repository can decode and use them.
*   **No Rotation**: Changing the password requires a manual commit to update the Secret.

**Verdict**: This is **NOT SAFE** for production environments unless the GitOps repository is strictly restricted, and even then, it breaks "Principle of Least Privilege".

## 2. Safer Alternatives
### A. Pre-Provisioned Secrets (Recommended for this Phase)
*   **Mechanism**: The platform administrator (you) creates a `docker-credentials` secret in the namespace *once* (as shown in the `walkthrough.md` prerequisites).
*   **Template**: The template just references `name: docker-credentials`.
*   **Pros**: Secure, credentials never touch the user's browser or the repo.
*   **Cons**: User cannot bring their *own*, personal credentials easily (requires admin help).

### B. External Secrets Operator (ESO)
*   **Mechanism**: Store credentials in a vault (AWS Secrets Manager, HashiCorp Vault). Use `ExternalSecret` CRD in Git.
*   **Pros**: GitOps friendly, no actual secrets in Git.
*   **Cons**: High infrastructure complexity to set up.

### C. Sealed Secrets
*   **Mechanism**: Encrypt the secret using a cluster-side public key. Commit the `SealedSecret` CRD.
*   **Pros**: Safe to commit to Git.
*   **Cons**: Requires `kubeseal` CLI or backend plugin integration to encrypt during scaffolding.

## 3. "Repository Name" Input
*   **Current**: `RepoUrlPicker` handles Host selection (github.com vs GHE) and Owner discovery (finding allowed Orgs).
*   **Proposed Simplification**: Replace with two text fields:
    1.  `githubOrg` (Default from config)
    2.  `repoName` (User types "my-app")
*   **Tradeoff**: Loss of auto-discovery of allowed organizations. User *must* know the correct Org name.

## Recommendation
1.  **Security**: Revert to the "Pre-provisioned Secret" model for the template default, but keep the *option* for custom credentials if the user explicitly understands the risk (or remove it to enforce security). **For this task, I will revert to the safe "Pre-provisioned" model as per your security concern.**
2.  **UX**: Implement the "Repo Name" text field to verify the UX improvement you requested.
