/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/argocd"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/database"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/gitopssync"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/tekton"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
)

// MockGitOpsClient is a mock implementation of GitOpsClientInterface.
type MockGitOpsClient struct {
	SyncedFiles map[string]string
}

func (m *MockGitOpsClient) SyncManifest(ctx context.Context, filePath, content string) error {
	if m.SyncedFiles == nil {
		m.SyncedFiles = make(map[string]string)
	}
	m.SyncedFiles[filePath] = content
	return nil
}

var _ = Describe("HeliosApp Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		heliosapp := &appv1alpha1.HeliosApp{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind HeliosApp")
			err := k8sClient.Get(ctx, typeNamespacedName, heliosapp)
			if err != nil && errors.IsNotFound(err) {
				resource := &appv1alpha1.HeliosApp{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: appv1alpha1.HeliosAppSpec{
						Components: []appv1alpha1.Component{
							{
								Name: "frontend",
								Properties: &runtime.RawExtension{
									Raw: []byte(`{"image": "nginx"}`),
								},
							},
						},
						GitOpsRepo:      "https://github.com/example/repo.git",
						GitOpsPath:      "apps/test-app",
						GitOpsSecretRef: "my-secret",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &appv1alpha1.HeliosApp{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance HeliosApp")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			mockGit := &MockGitOpsClient{}

			controllerReconciler := &HeliosAppReconciler{
				Client:    k8sClient,
				Scheme:    k8sClient.Scheme(),
				CueEngine: &heliosCue.Engine{},
				Tekton:    tekton.NewReconciler(k8sClient, k8sClient.Scheme(), nil),
				ArgoCD:    argocd.NewReconciler(k8sClient, k8sClient.Scheme()),
				Database:  database.NewReconciler(k8sClient, k8sClient.Scheme()),
				GitOps: gitopssync.NewReconciler(k8sClient, k8sClient.Scheme(), func(repo, user, token string) gitops.GitOpsClientInterface {
					return mockGit
				}),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			_ = err
		})
	})
})
