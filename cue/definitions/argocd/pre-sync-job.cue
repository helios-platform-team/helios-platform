// pre-sync-job.cue: ArgoCD PreSync Job template for database migrations
// This job runs before ArgoCD syncs the application, ensuring migrations complete successfully before deployment
package argocd

// #DatabaseMigrationPreSyncJob generates a Kubernetes Job with ArgoCD PreSync hook annotation
// Used by the operator to create a Job that runs before PostgREST pods are deployed
#DatabaseMigrationPreSyncJob: {
	// Input parameters
	appName:              string
	namespace:           string
	migrateImageRef:     string // e.g., "myorg/my-app-migrate:latest"
	databaseSecretRef:   string // Secret containing database credentials
	backoffLimit:        int    | *3
	ttlSecondsAfterFinished: int | *3600
	serviceAccountName:  string | *"\(appName)-migrator"

	// Output: Kubernetes Job object with PreSync hook
	output: {
		apiVersion: "batch/v1"
		kind:       "Job"
		metadata: {
			name:      "\(appName)-db-migrate-presync"
			namespace: namespace
			labels: {
				"app":              appName
				"job-type":         "db-migration"
				// ArgoCD hook annotations
				"argocd.argoproj.io/hook": "PreSync"
				"argocd.argoproj.io/hook-deletion-policy": "BeforeHookCreation"
			}
			annotations: {
				"argocd.argoproj.io/hook":               "PreSync"
				"argocd.argoproj.io/hook-deletion-policy": "BeforeHookCreation"
			}
		}
		spec: {
			backoffLimit:         backoffLimit
			ttlSecondsAfterFinished: ttlSecondsAfterFinished
			serviceAccountName:   serviceAccountName
			template: {
				metadata: {
					labels: {
						"app":      appName
						"job-type": "db-migration"
					}
				}
				spec: {
					// Run database migrations
					containers: [{
						name:            "db-migrate"
						image:           migrateImageRef
						imagePullPolicy: "Always"
						env: [
							{
								name: "PGRST_DB_URI"
								valueFrom: secretKeyRef: {
									name: databaseSecretRef
									key:  "uri"
								}
							},
						]
						resources: {
							requests: {
								cpu:    "100m"
								memory: "128Mi"
							}
							limits: {
								cpu:    "500m"
								memory: "512Mi"
							}
						}
						securityContext: {
							runAsNonRoot:             true
							runAsUser:                1000
							fsReadOnlyRootFilesystem: true
						}
					}]
					restartPolicy: "Never"
					securityContext: {
						runAsNonRoot: true
						runAsUser:    1000
					}
				}
			}
		}
	}
}

// #DatabaseMigrationServiceAccount generates ServiceAccount for migration Job
#DatabaseMigrationServiceAccount: {
	appName:   string
	namespace: string

	output: {
		apiVersion: "v1"
		kind:       "ServiceAccount"
		metadata: {
			name:      "\(appName)-migrator"
			namespace: namespace
			labels: {
				"app": appName
			}
		}
	}
}

// #DatabaseMigrationPostSyncJob generates a PostSync hook to restart PostgREST pods
// This ensures pods pull the latest migration image and schema changes take effect
#DatabaseMigrationPostSyncJob: {
	appName:              string
	namespace:           string
	postgreName:         string
	ttlSecondsAfterFinished: int | *600

	output: {
		apiVersion: "batch/v1"
		kind:       "Job"
		metadata: {
			name:      "\(appName)-postgrest-restart-postsync"
			namespace: namespace
			labels: {
				"app":              appName
				"job-type":         "pod-restart"
				"argocd.argoproj.io/hook": "PostSync"
				"argocd.argoproj.io/hook-deletion-policy": "BeforeHookCreation"
			}
			annotations: {
				"argocd.argoproj.io/hook":               "PostSync"
				"argocd.argoproj.io/hook-deletion-policy": "BeforeHookCreation"
			}
		}
		spec: {
			ttlSecondsAfterFinished: ttlSecondsAfterFinished
			template: {
				metadata: {
					labels: {
						"app":      appName
						"job-type": "pod-restart"
					}
				}
				spec: {
					// Restart PostgREST Deployment to pull latest migration image
					serviceAccountName: "\(appName)-migrator"
					containers: [{
						name:  "kubectl"
						image: "bitnami/kubectl:latest"
						command: [
							"kubectl",
							"rollout",
							"restart",
							"deployment/\(postgreName)",
							"-n",
							namespace,
						]
						resources: {
							requests: {
								cpu:    "50m"
								memory: "64Mi"
							}
							limits: {
								cpu:    "200m"
								memory: "256Mi"
							}
						}
					}]
					restartPolicy: "Never"
				}
			}
		}
	}
}
