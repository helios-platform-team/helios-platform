# ${{ values.name }} - PostGraphile GraphQL API

This is an instant, production-ready GraphQL API service scaffolded over PostgreSQL using PostGraphile, powered by the Helios Platform.

## What is PostGraphile?

PostGraphile dynamically reflects your underlying PostgreSQL database schema into a high-performance GraphQL API at runtime. Instead of writing schemas, resolvers, and boilerplate routing, you simply design your database tables and permissions in SQL—PostGraphile generates the complete CRUD GraphQL graph instantly.

## Key Features

- ⚡ **Instant GraphQL CRUD**: Full queries, mutations, connections, and relational filters derived directly from database schema.
- 🗲 **Zero-Downtime Migrations**: Evolve database schemas seamlessly via Git without pod restarts or service disruption.
- 🛡️ **Operator-Driven Secret Management**: Complete separation of database credentials with automatic token handling.
- 🎨 **Interactive GraphiQL Suite**: Built-in developer portal featuring documentation exploration and real-time query debugging.
- 🔄 **Full GitOps Pipeline**: Automated building via Tekton and deployment orchestration handled by ArgoCD.

---

## Architecture Flow

```text
Your Source Repo (Code/SQL)
            ↓ (On git push)
    Tekton CI/CD Pipeline 
            ├─→ Build: Custom Node/PostGraphile Image
            └─→ Push: docker.io/org/repo:main
            ↓
      ArgoCD Sync
            └─→ Roll out updated manifests
            ↓
    PostGraphile Pod (Running .postgraphilerc.js)
            ↓ (Bypasses Operator string expansion)
    PostgreSQL StatefulSet (Provisioned by Helios Operator)
```

## Folder Structure

This repository separates your application logic from database schema migration sets:

```plaintext
source-repo/
  ├── Dockerfile                    # Extends official graphile/postgraphile image
  ├── .postgraphilerc.js            # Main configuration suite (schemas, ports, watch)
  ├── catalog-info.yaml             # Backstage component model tracking
  ├── README.md                     # This documentation file
  └── db/
      └── migrations/               # Zero-downtime sequential migration set
          ├── 000001_initial_schema.up.sql
          └── 000001_initial_schema.down.sql
```

---

## 🚀 Getting Started & Local Testing

### Step 1: Port-Forward the GraphQL Service

To view your live service from your local developer workstation, forward your cluster service port to your host network:

```bash
kubectl port-forward svc/${{ values.name }} 8005:5000 -n ${{ values.namespace }}
```

### Step 2: Access the Endpoints

Once forwarded, PostGraphile exposes two foundational paths on your machine:

- **API Endpoint** (`/graphql`): http://localhost:8005/graphql *(For client applications)*
- **Visual Explorer Suite** (`/graphiql`): http://localhost:8005/graphiql *(For debugging in your browser)*

### Step 3: Executing Your First Test Query

Open http://localhost:8005/graphiql in your web browser. Paste the following schema query into the workspace panel and hit the Play button:

```graphql
query {
  allUsers {
    nodes {
      id
      username
      email
      createdAt
    }
  }
  allPosts {
    nodes {
      id
      title
      content
      userByAuthorId {
        username
      }
    }
  }
}
```

---

## ⚠️ Critical: The "Chicken and Egg" First-Time Scaffolding Quirk

When you first scaffold this template via the Backstage Portal, your PostGraphile pod will successfully spin up, but executing a query might yield a schema error stating: `"Cannot query field allUsers on type Query"`.

This occurs due to a known **GitOps Lifecycle Race Condition**:

1. **Repository Setup**: Backstage pushes your boilerplate files (with your initial SQL migrations) into Gitea.
2. **Webhook Hookup**: Backstage registers the Gitea repository webhooks seconds after that push finishes.

Because the webhook didn't exist at the exact microsecond the initial push occurred, Gitea never sends the activation event to Tekton. Consequently, your database spins up blank and empty.

### The Remediation Workflow

To easily prime your database on its first deployment, simply push any subsequent commit into your repository. This triggers your webhook, starts the Tekton `db-migrate` pipeline, and populates your baseline tables:

```bash
# Add an empty change or advance to your next feature column
echo "-- Trigger initialization" >> db/migrations/000001_initial_schema.up.sql

git add db/migrations/000001_initial_schema.up.sql
git commit -m "chore: kickstart database initialization pipeline"
git push origin main
```

---

## Day-to-Day Operations

### Modifying Schema Objects

To evolve your GraphQL endpoints, never overwrite older files inside `db/migrations/`. Instead, simply generate sequential migration pairs using leading numbers:

```bash
# 1. Create a new up/down pair
touch db/migrations/000002_add_profile_bio.up.sql
touch db/migrations/000002_add_profile_bio.down.sql

# 2. Write your schema alteration statement
echo "ALTER TABLE public.users ADD COLUMN bio TEXT;" >> db/migrations/000002_add_profile_bio.up.sql

# 3. Push to your main branch
git add db/migrations/
git commit -m "feat: add user bio column"
git push origin main
```

The underlying `db-migrate` execution network automatically applies the updates in the background, instantly updating your `/graphiql` schema tree with zero user downtime!