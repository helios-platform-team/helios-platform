# ${{ values.name }} - Hasura GraphQL API



This is the source repository for a Hasura GraphQL Engine instance, scaffolded by the Helios Portal. 



Hasura provides an instant, real-time GraphQL API over a PostgreSQL database. This repository contains the build instructions and configuration for your custom Hasura image.



---



## 🏗️ Architecture & Workflow



This project uses a fully automated **GitOps and CI/CD** workflow managed by the Helios Platform. 



```text

1. You commit code/metadata to THIS Source Repository

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

7. Hasura container rolls out, automatically connected to the database!

```


📂 Generated Repositories

-------------------------



This template generated **two** repositories for your application:



### 1\. The Source Repository (This Repo)



Contains your application build instructions:



*   Dockerfile: Builds your custom Hasura image. You can modify this to bake in Hasura migrations or metadata.

    

*   README.md: This documentation file.

    



### 2\. The GitOps Repository (\*-gitops)



Contains your Kubernetes infrastructure declarations:



*   helios-app.yaml: The Custom Resource that tells the Helios Operator how to deploy Hasura and provision the database.

    

*   catalog-info.yaml: Registers your component in the Backstage catalog.

    



🚀 Getting Started

------------------



### 1\. Accessing the Hasura Console



The Helios Operator automatically deploys your Hasura instance with the console enabled.Once your deployment is synced via ArgoCD, you can access the Hasura UI:



1.  Find your ingress URL: kubectl get ingress -n default (or your target namespace).

    

2.  Navigate to the URL in your browser to access the Hasura Console.

    

3.  From the console, you can track tables, set up relationships, and test GraphQL queries.

    



### 2\. Database Connection (Zero Configuration)



You **do not** need to manually configure the database connection!The Helios Operator automatically provisions a PostgreSQL database and natively injects the formatted connection string into your Hasura container using the HASURA\_GRAPHQL\_DATABASE\_URL environment variable.



### 3\. Customizing Your Hasura Image



If you want to track your database migrations and Hasura metadata in version control (recommended for production):



1.  Install the [Hasura CLI](https://www.google.com/search?q=https://hasura.io/docs/latest/hasura-cli/install-hasura-cli/).

    

2.  Initialize a Hasura project locally and track your changes.

    

3.  Update the Dockerfile in this repository to copy your migrations/ and metadata/ directories into the image.

    

4.  Commit and push! Tekton will automatically build the new image and roll it out.

    



🛠️ Troubleshooting

-------------------



**1\. The pipeline failed during the build phase:**Ensure your Docker Hub credentials are correct. The cluster requires a valid docker-credentials secret in the deployment namespace. The Helios Operator handles this automatically if configured with the right environment variables.



**2\. Hasura cannot connect to the database:**Check the logs of the Helios Operator to ensure the PostgreSQL instance was provisioned successfully. Verify that the HASURA\_GRAPHQL\_DATABASE\_URL is set to $(DATABASE\_URL) in your helios-app.yaml GitOps manifest.



**3\. Changes are not deploying:**Check ArgoCD to ensure the GitOps repository is syncing correctly. You can also monitor the Tekton PipelineRun in your cluster to see if the build and GitOps update steps completed successfully.



📚 Useful Links

---------------



*   [Hasura Official Documentation](https://www.google.com/search?q=https://hasura.io/docs/latest/index/)

    

*   [Learn GraphQL Basics](https://graphql.org/learn/)

    

*   [Helios Operator Documentation](https://www.google.com/search?q=../../../../docs/OPERATOR.md)