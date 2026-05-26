# ${{ values.name }} - Hasura GraphQL API



This is the source repository for a Hasura GraphQL Engine instance, scaffolded by the Helios Portal. 



Hasura provides an instant, real-time GraphQL API over a PostgreSQL database. This repository contains the build instructions, database migrations, and metadata configuration for your custom Hasura image.



---



## 🏗️ Architecture & Workflow



This project uses a fully automated **GitOps and CI/CD** workflow managed by the Helios Platform. By utilizing the official Hasura `cli-migrations-v3` image, your database schema and GraphQL API remain perfectly synchronized with your Git repository.



```text

1. You commit SQL migrations/metadata to THIS Source Repository

                ↓

2. Gitea Webhook triggers Tekton CI/CD Pipeline

                ↓

3. Tekton builds your custom Docker image and pushes to Docker Hub

                ↓

4. Tekton automatically updates the GitOps Repository with the new image tag

                ↓

5. ArgoCD detects the change and syncs the Kubernetes manifests

                ↓

6. Helios Operator provisions/updates the PostgreSQL Database 

                ↓

7. Hasura container rolls out, automatically applying your SQL migrations on startup!

```



📂 Generated Repositories

-------------------------



This template generated **two** repositories for your application:



### 1\. The Source Repository (This Repo)



Contains your application build instructions and database state:



*   Dockerfile: Builds your custom Hasura image. It is pre-configured to copy your migrations into the container.

    

*   migrations/: Contains your raw SQL files. A base migration (init) is already provided.

    

*   metadata/: Tracks your Hasura API configuration (table tracking, relationships, permissions).

    

*   README.md: This documentation file.

    



### 2\. The GitOps Repository (\*-gitops)



Contains your Kubernetes infrastructure declarations:



*   helios-app.yaml: The Custom Resource that tells the Helios Operator how to deploy Hasura and natively wire up the PostgreSQL database connection.

    

*   catalog-info.yaml: Registers your component in the Backstage catalog.

    



🚀 Getting Started

------------------



### 1\. Accessing the Hasura Console



The Helios Operator automatically deploys your Hasura instance with the console enabled. To access it locally during development:



1.  Run kubectl port-forward svc/${{ values.name }} 8085:8080 -n default (update the namespace if needed).

    

2.  Open your browser and navigate to http://localhost:8085/console.

    



### 2\. The Initial Database State



We have pre-populated your database with a base migration!



1.  In the Hasura Console, click the **DATA** tab.

    

2.  Expand the default database and click the public schema.

    

3.  Under **Untracked tables or views**, you will see a users table.

    

4.  Click **Track** to expose this table to your GraphQL API.

    



🛠️ The GitOps Migration Workflow

---------------------------------



To manage your database schema, you do not need external tools. You just write SQL and push!



### Adding a New Table



1.  mkdir -p migrations/default/1700000000001\_create\_posts/

    

2.  SQLCREATE TABLE public.posts ( id SERIAL PRIMARY KEY, title TEXT NOT NULL, content TEXT);

    

3.  Commit and push the code.

    

4.  The Tekton pipeline will automatically build the new Docker image, ArgoCD will roll it out, and Hasura will automatically execute your new SQL file against the database!

    

5.  Open the Hasura Console to track your new table.

    



⚙️ Managing Metadata (Crucial Step)

-----------------------------------



"Metadata" is how Hasura remembers which tables are exposed to the GraphQL API, your relationships, and your security permissions.



**Currently, metadata copying is DISABLED in your Dockerfile so the initial deployment doesn't crash on an empty configuration.**



Once you track your first tables in the Hasura UI, you need to save that configuration back to this Git repository:



1.  Install the [Hasura CLI](https://www.google.com/search?q=https://hasura.io/docs/latest/hasura-cli/install-hasura-cli/).

    

2.  Initialize a Hasura project locally or connect to your running instance.

    

3.  Run hasura metadata export to generate the .yaml files in your metadata/ directory.

    

4.  DockerfileRUN mkdir -p /hasura-metadataCOPY metadata/ /hasura-metadata/

    

5.  Commit and push. From now on, your API configuration is strictly version-controlled!

    



🚑 Troubleshooting

------------------



**1\. The pipeline didn't trigger when I pushed code:**Ensure the Gitea Webhook is configured correctly in your repository settings to point to the Tekton EventListener.



**2\. Hasura cannot connect to the database (CrashLoopBackOff):**Check the logs of the Helios Operator. Verify that the database was provisioned. Hasura relies on the explicit env variable mapping in your GitOps helios-app.yaml to construct the HASURA\_GRAPHQL\_DATABASE\_URL from the Operator's injected secrets.



**3\. Hasura crashes on startup with a parse-failed metadata error:**This means you uncommented the metadata COPY step in the Dockerfile, but your metadata/ folder doesn't contain valid Hasura configuration files yet. Comment it back out until you have exported your metadata via the CLI.



📚 Useful Links

---------------



*   [Hasura Official Documentation](https://www.google.com/search?q=https://hasura.io/docs/latest/index/)

    

*   [Learn GraphQL Basics](https://graphql.org/learn/)

    

*   [Helios Operator Documentation](https://www.google.com/search?q=../../../../docs/OPERATOR.md)

