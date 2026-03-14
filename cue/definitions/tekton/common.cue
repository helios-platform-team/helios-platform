// Shared definitions used across all Tekton resources.
// Single source of truth for defaults, common parameters, and labels.
package tekton

// =====================================================
// DEFAULTS: Centralized configuration values
// Change here = change everywhere
// =====================================================
#Defaults: {
	// Container images - PIN VERSION for reproducibility
	images: {
		gitClone: "alpine/git:v2.43.0"
		kaniko:   "gcr.io/kaniko-project/executor:v1.19.2"
		alpine:   "alpine:3.19"
		yq:       "mikefarah/yq:4.40.5"
	}

	// Secret names
	secrets: {
		docker: "docker-credentials"
		github: "github-credentials"
	}

	// Tekton API versions
	tekton: {
		apiVersion:     "tekton.dev/v1beta1"
		triggerVersion: "triggers.tekton.dev/v1beta1"
		serviceAccount: "tekton-triggers-sa"
	}

	// Workspace names
	workspaces: {
		source:         "source-workspace"
		gitops:         "gitops-workspace"
		gitCredentials: "git-credentials-workspace"
	}
}

// =====================================================
// COMMON PARAMS: Reusable parameter definitions
// Ensures consistency across tasks/pipelines
// =====================================================
#CommonParams: {
	// Git source params
	git: {
		url: {
			name:        "url"
			description: "Repository URL to clone"
			type:        "string"
		}
		revision: {
			name:        "revision"
			description: "Git revision (branch, tag, or commit SHA)"
			type:        "string"
			default:     "main"
		}
	}

	// Image params for Kaniko
	image: {
		name: {
			name:        "IMAGE"
			description: "Full image name with registry"
			type:        "string"
		}
		dockerfile: {
			name:    "DOCKERFILE"
			type:    "string"
			default: "Dockerfile"
		}
		contextSubpath: {
			name:        "CONTEXT_SUBPATH"
			description: "Subdirectory where Dockerfile is located"
			type:        "string"
			default:     ""
		}
		dockerSecret: {
			name:        "docker-secret"
			description: "Name of secret with Docker credentials"
			type:        "string"
			default:     "docker-credentials"
		}
	}

	// GitOps params for manifest updates
	gitops: {
		repoUrl: {
			name:        "GITOPS_REPO_URL"
			description: "GitOps repository URL"
			type:        "string"
		}
		branch: {
			name:    "GITOPS_REPO_BRANCH"
			type:    "string"
			default: "main"
		}
		manifestPath: {
			name:        "MANIFEST_PATH"
			description: "Path to manifest file in GitOps repo"
			type:        "string"
		}
		newImageUrl: {
			name:        "NEW_IMAGE_URL"
			description: "New image URL to update in manifest"
			type:        "string"
		}
		secret: {
			name:    "GITOPS_SECRET"
			type:    "string"
			default: "github-credentials"
		}
	}

	// App params
	app: {
		name: {
			name:        "app-name"
			description: "Application name"
			type:        "string"
		}
		repoUrl: {
			name:        "app-repo-url"
			description: "Source repository URL"
			type:        "string"
		}
		repoRevision: {
			name:        "app-repo-revision"
			description: "Source repository revision"
			type:        "string"
			default:     "main"
		}
		imageRepo: {
			name:        "image-repo"
			description: "Container image repository"
			type:        "string"
		}
	}

	// Testing params
	test: {
		command: {
			name:        "test-command"
			description: "Command to run tests"
			type:        "string"
			default:     ""
		}
		image: {
			name:        "test-image"
			description: "Image to use for running tests"
			type:        "string"
			default:     "node:20"
		}
	}
}

// =====================================================
// LABELS: Applied to all generated resources
// =====================================================
#CommonLabels: {
	"helios.io/managed-by":       "helios-operator"
	"app.kubernetes.io/part-of":  "helios-platform"
	"app.kubernetes.io/instance": string | *"default"
	"app.kubernetes.io/name"?:    string
	... // Allow additional labels
}

// Helper to generate labels with app name
#AppLabels: {
	_appName: string
	labels: #CommonLabels & {
		"app.kubernetes.io/instance": _appName
		"app.kubernetes.io/name":     _appName
	}
}
