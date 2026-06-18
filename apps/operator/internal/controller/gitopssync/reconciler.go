package gitopssync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/provider"
)

// GitFactory creates a GitOps client for the given repo, user, and token.
type GitFactory func(repo, user, token string) gitops.ClientInterface

// Reconciler handles GitOps manifest sync (credentials + git push + hash).
type Reconciler struct {
	Client      client.Client
	Scheme      *runtime.Scheme
	GitFactory  GitFactory
	Credentials provider.CredentialsResolver
}

// NewReconciler creates a new GitOps sync Reconciler.
func NewReconciler(c client.Client, scheme *runtime.Scheme, factory GitFactory) *Reconciler {
	if factory == nil {
		factory = func(repo, user, token string) gitops.ClientInterface {
			return gitops.NewClient(repo, user, token)
		}
	}
	return &Reconciler{
		Client:      c,
		Scheme:      scheme,
		GitFactory:  factory,
		Credentials: provider.NewCredentialResolver(c, provider.Default),
	}
}

// Reconcile resolves GitOps credentials, computes a manifest hash, and syncs
// changed manifests to the GitOps repo. It also updates the HeliosApp status.
func (r *Reconciler) Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp, manifestBytes []byte) error {
	log := logf.FromContext(ctx)

	token, username, err := r.Credentials.ResolveGitCredentials(ctx, app.Namespace, app.Spec.GitOpsSecretRef)
	if err != nil {
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
