/*
Copyright 2025.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	heliosappv1 "github.com/hoangphuc841/helios-operator/api/v1"
	"github.com/hoangphuc841/helios-operator/internal/engine"
	"github.com/hoangphuc841/helios-operator/internal/gitops"
	"k8s.io/client-go/tools/record"
)

// HeliosAppReconciler reconciles a HeliosApp object
type HeliosAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Thêm EventRecorder để bắn event ra K8s
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=platform.helios.io,resources=heliosapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.helios.io,resources=heliosapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.helios.io,resources=heliosapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=triggers.tekton.dev,resources=eventlisteners;triggerbindings;triggertemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=pods;podmetrics,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;update;patch;delete

func (r *HeliosAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// ========================================================================
	// BƯỚC 1: Lấy HeliosApp instance
	// ========================================================================
	var heliosApp heliosappv1.HeliosApp
	if err := r.Get(ctx, req.NamespacedName, &heliosApp); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("HeliosApp not found. It may have been deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Unable to fetch HeliosApp")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	name := heliosApp.Name
	namespace := heliosApp.Namespace

	// ========================================================================
	// BƯỚC 4: Tự động tối ưu hóa tài nguyên (Smart Optimization) & Lấy Metrics
	// ========================================================================
	logger.Info("DEBUG: Checking Ready condition", "conditions", heliosApp.Status.Conditions)
	logger.Info("DEBUG: Checking Ready condition", "conditions", heliosApp.Status.Conditions)
	isReady := false
	for _, c := range heliosApp.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			isReady = true
			break
		}
	}

	if isReady {
		logger.Info("DEBUG: Conditions met! Calling Optimizer...")
		optimizer := &OptimizerService{}
		optResult := optimizer.AnalyzeResourceUsage(ctx, r.Client, &heliosApp)

		// Cập nhật Metrics vào Status để Frontend hiển thị
		if optResult.MeasuredCpu != "" {
			heliosApp.Status.CurrentCPU = optResult.MeasuredCpu
		}

		// Save Status
		if err := r.Status().Update(ctx, &heliosApp); err != nil {
			logger.Error(err, "Failed to update HeliosApp status with metrics")
		}

		if heliosApp.Spec.EnableAutoOptimization && optResult.IsWasteful {
			logger.Info("Waste detected! Triggering auto-fix...", "waste", optResult.WastePercentage)

			// Thực hiện Auto-Fix (GitOps Write-Back)
			if err := optimizer.AutoFixRepository(ctx, &heliosApp, optResult); err != nil {
				logger.Error(err, "Failed to auto-fix repository")
				r.EventRecorder.Event(&heliosApp, corev1.EventTypeWarning, "AutoFixFailed", err.Error())
			} else {
				r.EventRecorder.Event(&heliosApp, corev1.EventTypeNormal, "AutoFixTriggered", "Created PR to optimize resources")
			}
		}
	}
	// ========================================================================
	// PHASE 0: Setup CI/CD Triggers (Tekton)
	// ========================================================================
	// Restore Tekton Logic: Define resource names
	triggerBindingName := name + "-git-binding"
	defaultsBindingName := name + "-defaults-binding"
	triggerTemplateName := name + "-template"
	eventListenerName := name + "-listener"
	pipelineName := heliosApp.Spec.PipelineName
	serviceAccount := heliosApp.Spec.ServiceAccount

	// 1. Create/Update TriggerBinding (Git Info)
	tbGit, _ := GenerateTriggerBinding(triggerBindingName, namespace)
	// Set owner ref
	if err := ctrl.SetControllerReference(&heliosApp, tbGit, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for TriggerBinding (Git)")
	}
	// Apply (Create if not exists)
	ctxBg := context.Background() // Use background context for resource creation to avoid timeout issues
	if err := r.Client.Create(ctxBg, tbGit); err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create TriggerBinding (Git)")
			return ctrl.Result{}, err
		}
		// If exists, strictly we should update, but for now assume static
	}

	// 2. Create/Update TriggerBinding (Defaults)
	tbDefaults, _ := GenerateDefaultsTriggerBinding(defaultsBindingName, namespace, &heliosApp)
	if err := ctrl.SetControllerReference(&heliosApp, tbDefaults, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for TriggerBinding (Defaults)")
	}
	if err := r.Client.Create(ctxBg, tbDefaults); err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create TriggerBinding (Defaults)")
			return ctrl.Result{}, err
		}
	}

	// 3. Create/Update TriggerTemplate
	// Workspace map is just a placeholder here because the helper constructs it internally
	workspaceConfig := map[string]any{}
	tt, _ := GenerateTriggerTemplate(triggerTemplateName, namespace, name+"-run", pipelineName, serviceAccount, workspaceConfig)
	if err := ctrl.SetControllerReference(&heliosApp, tt, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for TriggerTemplate")
	}
	if err := r.Client.Create(ctxBg, tt); err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create TriggerTemplate")
			return ctrl.Result{}, err
		}
	}

	// 4. Create/Update EventListener
	// Trigger Name: "github-push"
	el, _ := GenerateEventListener(eventListenerName, namespace, "github-push", triggerBindingName, defaultsBindingName, triggerTemplateName, heliosApp.Spec.WebhookSecret)
	if err := ctrl.SetControllerReference(&heliosApp, el, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for EventListener")
	}
	if err := r.Client.Create(ctxBg, el); err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create EventListener")
			return ctrl.Result{}, err
		}
	}

	logger.Info("CI/CD Triggers setup completed", "EventListener", eventListenerName)

	logger.Info("Reconciling HeliosApp", "name", name, "namespace", namespace, "generation", heliosApp.Generation)

	// ========================================================================
	// PHASE 1: Render & GitOps (Direct Write) - REPLACED TEKTON
	// ========================================================================

	// Check if we need to sync (Optimization: Check generation or force sync)
	// For simplicity, we sync if ObservedGeneration != Generation

	if heliosApp.Status.ObservedGeneration != heliosApp.Generation {
		logger.Info("Spec changed, executing GitOps Sync", "generation", heliosApp.Generation)

		// 1. Render Manifest
		logger.Info("Rendering manifests...")
		manifestContent, err := engine.RenderManifest(&heliosApp)
		if err != nil {
			logger.Error(err, "Failed to render manifest")
			// Update status to Failed
			updateStatus := heliosApp.DeepCopy()
			updateStatus.Status.Phase = "RenderFailed"
			updateStatus.Status.Message = err.Error()
			r.Status().Update(ctx, updateStatus)
			return ctrl.Result{}, err
		}

		// 2. Fetch Git Credentials (from Secret)
		// We expect a secret named by Spec.WebhookSecret (reuse field for simplicity) or hardcoded env
		githubToken := ""

		// Try secret
		secretName := heliosApp.Spec.WebhookSecret
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err == nil {
			if tokenBytes, ok := secret.Data["secretToken"]; ok {
				githubToken = string(tokenBytes)
			}
		}

		// Fallback to env
		if githubToken == "" {
			githubToken = os.Getenv("GITHUB_TOKEN")
			if githubToken == "" {
				// Final fallback for local dev if env is missing
				logger.Info("Warning: No token found in secret or env, Git push might fail if auth is required")
			}
		}

		// 3. Push to Git
		logger.Info("Pushing to GitOps Repo", "repo", heliosApp.Spec.GitOpsRepo)
		gitClient := gitops.NewGitOpsClient(heliosApp.Spec.GitOpsRepo, "git", githubToken)

		// Target path: e.g., "apps/production/frontend/manifest.yaml"
		targetPath := fmt.Sprintf("%s/manifest.yaml", heliosApp.Spec.GitOpsPath)

		if err := gitClient.SyncManifest(ctx, targetPath, manifestContent); err != nil {
			logger.Error(err, "GitOps Sync Failed")

			updateStatus := heliosApp.DeepCopy()
			updateStatus.Status.Phase = "GitOpsFailed"
			updateStatus.Status.Message = fmt.Sprintf("Failed to push to git: %v", err)
			r.Status().Update(ctx, updateStatus)

			return ctrl.Result{}, err
		}

		logger.Info("GitOps Sync Successful!")

		// 4. Update Status to trigger ArgoCD check next
		heliosApp.Status.ObservedGeneration = heliosApp.Generation
		heliosApp.Status.Phase = "ManifestGenerated"
		heliosApp.Status.Message = "Manifest pushed to GitOps repo. Waiting for ArgoCD."

		if err := r.Status().Update(ctx, &heliosApp); err != nil {
			logger.Error(err, "Failed to update status after GitOps sync")
			return ctrl.Result{}, err
		}

		// Requeue immediately to proceed to Phase 2 (ArgoCD)
		return ctrl.Result{Requeue: true}, nil
	}

	// ========================================================================
	// GIAI ĐOẠN 3: Đồng bộ trạng thái từ ArgoCD Application
	// ========================================================================

	var syncStatus, healthStatus string
	if heliosApp.Status.ArgoApplication != "" {
		argoApp := &unstructured.Unstructured{}
		argoApp.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "argoproj.io",
			Version: "v1alpha1",
			Kind:    "Application",
		})

		err := r.Get(ctx, types.NamespacedName{
			Name:      heliosApp.Status.ArgoApplication,
			Namespace: "argocd",
		}, argoApp)

		if err != nil {
			if errors.IsNotFound(err) {
				logger.Info("ArgoCD Application not found", "Application", heliosApp.Status.ArgoApplication)
				heliosApp.Status.Phase = "ArgoCDAppNotFound"
				heliosApp.Status.Message = fmt.Sprintf("ArgoCD Application %s not found. Recreating.", heliosApp.Status.ArgoApplication)
				heliosApp.Status.ArgoApplication = "" // Reset to trigger recreation
				r.Status().Update(ctx, &heliosApp)
				return ctrl.Result{Requeue: true}, nil
			} else {
				logger.Error(err, "Failed to get ArgoCD Application")
				// UPDATE STATUS: Failed
				heliosApp.Status.Phase = "Failed"
				heliosApp.Status.Message = fmt.Sprintf("ArgoCD Error: %v", err)
				r.Status().Update(ctx, &heliosApp)
				return ctrl.Result{}, err
			}
		} else {
			// Đọc status từ ArgoCD Application - dùng type assertion thay vì NestedMap để tránh panic
			statusRaw, statusExists := argoApp.Object["status"]
			if !statusExists {
				logger.Info("ArgoCD Application status not available yet")
				heliosApp.Status.Phase = "ArgoCDSyncing"
				heliosApp.Status.Message = fmt.Sprintf("ArgoCD Application %s status not available yet.", argoApp.GetName())
				r.Status().Update(ctx, &heliosApp)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			status, ok := statusRaw.(map[string]interface{})
			if !ok {
				logger.Info("ArgoCD Application status format unexpected")
				heliosApp.Status.Phase = "ArgoCDSyncing"
				heliosApp.Status.Message = fmt.Sprintf("ArgoCD Application %s status format unexpected.", argoApp.GetName())
				r.Status().Update(ctx, &heliosApp)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			// Lấy sync status
			syncStatus, _, _ = unstructured.NestedString(status, "sync", "status")
			healthStatus, _, _ = unstructured.NestedString(status, "health", "status")

			logger.Info("ArgoCD Application status",
				"Application", argoApp.GetName(),
				"syncStatus", syncStatus,
				"healthStatus", healthStatus)

			// Lấy deployed version (image từ summary hoặc revision)
			images, found, _ := unstructured.NestedSlice(status, "summary", "images")
			if found && len(images) > 0 {
				heliosApp.Status.DeployedVersion = images[0].(string)
			}

			// Cập nhật condition dựa trên trạng thái ArgoCD
			if syncStatus == "Synced" && healthStatus == "Healthy" {
				meta.SetStatusCondition(&heliosApp.Status.Conditions, metav1.Condition{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					Reason:             "SyncedAndHealthy",
					Message:            fmt.Sprintf("Application is synced and healthy. Deployed version: %s", heliosApp.Status.DeployedVersion),
					ObservedGeneration: heliosApp.Generation,
				})

				logger.Info("ArgoCD application is synced and healthy. Reconciliation complete.",
					"Application", argoApp.GetName(),
					"version", heliosApp.Status.DeployedVersion)
			} else {
				// Nếu ArgoCD chưa Synced, có thể cần trigger refresh để pull manifest mới
				if syncStatus == "OutOfSync" {
					logger.Info("ArgoCD Application is OutOfSync, triggering refresh", "Application", argoApp.GetName())
					if err := r.refreshArgoApplication(ctx, argoApp.GetName(), "argocd"); err != nil {
						logger.Error(err, "Failed to trigger ArgoCD refresh for OutOfSync app")
					}
				}

				meta.SetStatusCondition(&heliosApp.Status.Conditions, metav1.Condition{
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					Reason:             "Syncing",
					Message:            fmt.Sprintf("ArgoCD sync: %s, health: %s", syncStatus, healthStatus),
					ObservedGeneration: heliosApp.Generation,
				})
			}

			if err := r.Status().Update(ctx, &heliosApp); err != nil {
				logger.Error(err, "Failed to update status from ArgoCD")
				return ctrl.Result{}, err
			}

			// Tiếp tục theo dõi nếu chưa Synced và Healthy
			if syncStatus != "Synced" || healthStatus != "Healthy" {
				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}
		}
	}

	// Helper to update status safely
	updateStatus := func(phase, msg string) {
		heliosApp.Status.Phase = phase
		heliosApp.Status.Message = msg
		// We ignore error here to not block flow, mostly just for visibility
		if err := r.Status().Update(ctx, &heliosApp); err != nil {
			logger.Error(err, "Failed to update HeliosApp status in updateStatus helper")
		}
	}

	// Update Healthy Status if we reached here
	isHealthy := meta.IsStatusConditionTrue(heliosApp.Status.Conditions, "Ready")
	if isHealthy {
		updateStatus("Healthy", "App is running and synced")
	} else {
		updateStatus("Syncing", fmt.Sprintf("ArgoCD Status: %s/%s", syncStatus, healthStatus))
	}

	// ------------------------------------------------------------------
	// PHASE 4: Helios Smart Rightsizer (Auto-Optimization)
	// ------------------------------------------------------------------
	// Đây là tính năng "Killer" của đồ án: Tự động tối ưu tài nguyên

	// Chỉ chạy tối ưu khi App đã Healthy và User đã BẬT tính năng này
	isOptIn := heliosApp.Spec.EnableAutoOptimization

	if isHealthy && isOptIn {
		logger.Info("Auto-Optimization Condition Met", "isHealthy", isHealthy, "isOptIn", isOptIn)
		optimizer := &OptimizerService{}
		analysis := optimizer.AnalyzeResourceUsage(ctx, r.Client, &heliosApp)

		if analysis.IsWasteful {
			logger.Info("DETECTED RESOURCE WASTE",
				"app", heliosApp.Name,
				"current", analysis.CurrentCpu,
				"suggested", analysis.SuggestedCpu,
				"waste_percent", analysis.WastePercentage)

			// Execute Auto-Fix (Closed Loop GitOps)
			err := optimizer.AutoFixRepository(ctx, &heliosApp, analysis)
			if err != nil {
				logger.Error(err, "Failed to auto-optimize repository")
				updateStatus("OptimizationFailed", fmt.Sprintf("Could not create PR: %v", err))
			} else {
				// Success
				r.EventRecorder.Event(&heliosApp, corev1.EventTypeNormal, "AutoOptimized",
					fmt.Sprintf("Reduced CPU from %s to %s via GitOps", analysis.CurrentCpu, analysis.SuggestedCpu))
				updateStatus("Optimized", fmt.Sprintf("Waste detected (%d%%). PR created to fix.", analysis.WastePercentage))
			}
			if err != nil {
				logger.Error(err, "Failed to auto-optimize repository")
				// Không return error để tránh crash loop reconcile chính, chỉ log error
			} else {
				// Nếu fix thành công, update Event để thông báo cho user
				r.EventRecorder.Event(&heliosApp, corev1.EventTypeNormal, "AutoOptimized",
					fmt.Sprintf("Reduced CPU from %s to %s via GitOps", analysis.CurrentCpu, analysis.SuggestedCpu))
			}
		}
	}

	logger.Info("Reconciliation loop completed successfully for HeliosApp", "name", name)
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// Hàm so sánh spec của hai unstructured (chỉ so sánh phần spec)
// Sử dụng JSON marshal toàn bộ object để tránh panic với nested structures
func equalUnstructured(a, b *unstructured.Unstructured) bool {
	// Marshal toàn bộ object thay vì chỉ spec để tránh deep copy issues
	jsonA, errA := json.Marshal(a.Object)
	jsonB, errB := json.Marshal(b.Object)
	if errA != nil || errB != nil {
		// Nếu không marshal được, coi như khác nhau để update
		return false
	}

	return string(jsonA) == string(jsonB)
}

// refreshArgoApplication triggers a refresh operation on the ArgoCD Application
// This forces ArgoCD to pull the latest manifests from the GitOps repository
func (r *HeliosAppReconciler) refreshArgoApplication(ctx context.Context, appName, namespace string) error {
	logger := log.FromContext(ctx)

	// Get the ArgoCD Application
	argoApp := &unstructured.Unstructured{}
	argoApp.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := r.Get(ctx, types.NamespacedName{
		Name:      appName,
		Namespace: namespace,
	}, argoApp)

	if err != nil {
		return err
	}

	// Add refresh annotation to trigger ArgoCD to pull latest changes
	annotations := argoApp.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Use current timestamp to ensure annotation changes
	annotations["argocd.argoproj.io/refresh"] = "hard"
	annotations["helios.io/refreshed-at"] = time.Now().Format(time.RFC3339)
	argoApp.SetAnnotations(annotations)

	if err := r.Update(ctx, argoApp); err != nil {
		return err
	}

	logger.Info("Triggered ArgoCD refresh via annotation", "Application", appName)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HeliosAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&heliosappv1.HeliosApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// Note: Deployment and Service creation is handled by ArgoCD via GitOps manifests.
// If you need the operator to create K8s resources directly, reintroduce helper
// methods and call them from the reconciliation loop.
