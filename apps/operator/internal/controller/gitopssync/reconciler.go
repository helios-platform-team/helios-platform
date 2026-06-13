package gitopssync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
)

// GitFactory creates a GitOps client for the given repo, user, and token.
type GitFactory func(repo, user, token string) gitops.ClientInterface

// Reconciler handles GitOps manifest sync (credentials + git push + hash).
type Reconciler struct {
	Client     client.Client
	Scheme     *runtime.Scheme
	GitFactory GitFactory
}

// NewReconciler creates a new GitOps sync Reconciler.
func NewReconciler(c client.Client, scheme *runtime.Scheme, factory GitFactory) *Reconciler {
	if factory == nil {
		factory = func(repo, user, token string) gitops.ClientInterface {
			return gitops.NewClient(repo, user, token)
		}
	}
	return &Reconciler{Client: c, Scheme: scheme, GitFactory: factory}
}

// Reconcile resolves GitOps credentials, computes a manifest hash, and syncs
// changed manifests to the GitOps repo. It also updates the HeliosApp status.
func (r *Reconciler) Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp, manifestBytes []byte) error {
	log := logf.FromContext(ctx)

	token, username := r.resolveCredentials(ctx, app)

	if token == "" {
		err := fmt.Errorf("GitOps token is empty. Check Secret or GITEA_TOKEN env var")
		log.Error(err, "Authentication failed")
		r.updateFailedStatus(ctx, app, "GitOps token missing")
		return nil
	}

	currentHash := computeHash([]byte(app.Spec.GitOpsRepo + "\x00" + app.Spec.GitOpsPath + "\x00" + string(manifestBytes)))
	if app.Status.LastAppliedHash == currentHash {
		log.Info("Manifest hash unchanged, skipping GitOps sync", "hash", currentHash)

		if app.Status.Phase != appv1alpha1.PhaseReady {
			app.Status.Phase = appv1alpha1.PhaseReady
			if err := r.Client.Status().Update(ctx, app); err != nil {
				return err
			}
		}
		return nil
	}

	gitClient := r.GitFactory(app.Spec.GitOpsRepo, username, token)
	targetPath := fmt.Sprintf("%s/manifest.yaml", app.Spec.GitOpsPath)

	if err := gitClient.SyncManifest(ctx, targetPath, string(manifestBytes)); err != nil {
		log.Error(err, "GitOps sync failed")
		r.updateFailedStatus(ctx, app, fmt.Sprintf("GitOps failed: %v", err))
		return err
	}

	app.Status.Phase = appv1alpha1.PhaseReady
	app.Status.Message = fmt.Sprintf("Manifest pushed to %s/%s", app.Spec.GitOpsRepo, targetPath)
	app.Status.LastAppliedHash = currentHash
	app.Status.ResourcesCreated = nil

	if err := r.Client.Status().Update(ctx, app); err != nil {
		log.Error(err, "Failed to update status")
		return err
	}
	log.Info("Successfully reconciled HeliosApp via GitOps", "newHash", currentHash)

	return nil
}

// resolveCredentials reads the GitOps token and username from a Secret or env var.
func (r *Reconciler) resolveCredentials(ctx context.Context, app *appv1alpha1.HeliosApp) (string, string) {
	log := logf.FromContext(ctx)

	token := os.Getenv("GITEA_TOKEN")
	username := os.Getenv("GITEA_USER")
	if username == "" {
		username = "git"
	}

	if app.Spec.GitOpsSecretRef != "" {
		var secret corev1.Secret
		if err := r.Client.Get(ctx, types.NamespacedName{Name: app.Spec.GitOpsSecretRef, Namespace: app.Namespace}, &secret); err == nil {
			if t, ok := secret.Data["token"]; ok {
				token = string(t)
			} else if p, ok := secret.Data["password"]; ok {
				token = string(p)
			} else {
				log.Info("Secret found but 'token' or 'password' key is missing", "Secret", app.Spec.GitOpsSecretRef)
				r.updateFailedStatus(ctx, app, fmt.Sprintf("Secret %s missing 'token' key", app.Spec.GitOpsSecretRef))
				return "", ""
			}
			if u, ok := secret.Data["username"]; ok {
				username = string(u)
			}
		} else {
			log.Error(err, "Failed to get GitOps Secret", "Secret", app.Spec.GitOpsSecretRef)
			r.updateFailedStatus(ctx, app, fmt.Sprintf("Secret %s not found", app.Spec.GitOpsSecretRef))
			return "", ""
		}
	}

	return token, username
}

func (r *Reconciler) updateFailedStatus(ctx context.Context, app *appv1alpha1.HeliosApp, message string) {
	app.Status.Phase = appv1alpha1.PhaseFailed
	app.Status.Message = message
	if err := r.Client.Status().Update(ctx, app); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to update status")
	}
}

func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
