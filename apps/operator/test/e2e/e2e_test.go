/*
Copyright 2025.

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

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hoangphuc841/helios-operator/test/utils"
)

var _ = Describe("HeliosApp GitOps Flow E2E", Ordered, func() {
	const (
		testNamespace = "e2e-test"
		testAppName   = "test-gitops-app"
		timeout       = 5 * time.Minute
		interval      = 2 * time.Second
	)

	BeforeAll(func() {
		By("Creating test namespace")
		cmd := exec.Command("kubectl", "create", "namespace", testNamespace)
		_, _ = utils.Run(cmd) // Ignore error if already exists

		By("Creating ServiceAccount for Tekton")
		cmd = exec.Command("kubectl", "create", "serviceaccount", "tekton-bot-sa", "-n", testNamespace)
		_, _ = utils.Run(cmd)

		By("Installing mock manifest-generation-pipeline")
		cmd = exec.Command("kubectl", "apply", "-f", "../fixtures/mock-manifest-pipeline.yaml", "-n", testNamespace)
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to install mock pipeline: %s", output))
	})

	AfterAll(func() {
		By("Cleaning up test namespace")
		cmd := exec.Command("kubectl", "delete", "namespace", testNamespace, "--wait=false")
		_, _ = utils.Run(cmd)
	})

	Describe("3-Phase GitOps Reconciliation", func() {
		var heliosAppYAML string

		BeforeEach(func() {
			heliosAppYAML = fmt.Sprintf(`
apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: %s
  namespace: %s
spec:
  gitRepo: https://github.com/example/test-app
  gitBranch: main
  imageRepo: ghcr.io/example/test-app
  port: 8080
  replicas: 2
  serviceAccount: tekton-bot-sa
  pipelineName: manifest-generation-pipeline
  pvcName: workspace-pvc
  webhookSecret: github-webhook
  templateRepo: https://github.com/example/templates
  templatePath: charts/app
  gitopsRepo: https://github.com/example/gitops
  gitopsPath: apps/test
  values:
    env: test
    ingress.enabled: "false"
`, testAppName, testNamespace)
		})

		AfterEach(func() {
			By("Cleaning up HeliosApp")
			cmd := exec.Command("kubectl", "delete", "heliosapp", testAppName, "-n", testNamespace, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)

			By("Cleaning up Tekton resources")
			cmd = exec.Command("kubectl", "delete", "pipelinerun", "--all", "-n", testNamespace)
			_, _ = utils.Run(cmd)

			cmd = exec.Command("kubectl", "delete", "eventlistener", "--all", "-n", testNamespace)
			_, _ = utils.Run(cmd)
		})

		It("should complete Phase 1: Tekton Triggers creation", func() {
			By("Creating HeliosApp")
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(heliosAppYAML)
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create HeliosApp: %s", output))

			By("Waiting for EventListener to be created")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "eventlistener", "-n", testNamespace, "-o", "name")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(ContainSubstring("eventlistener"))

			By("Verifying TriggerBinding is created")
			cmd = exec.Command("kubectl", "get", "triggerbinding", fmt.Sprintf("%s-binding", testAppName), "-n", testNamespace)
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("TriggerBinding not found: %s", output))

			By("Verifying TriggerTemplate is created")
			cmd = exec.Command("kubectl", "get", "triggertemplate", fmt.Sprintf("%s-template", testAppName), "-n", testNamespace)
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("TriggerTemplate not found: %s", output))

			By("Checking HeliosApp status shows Phase 1")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "heliosapp", testAppName, "-n", testNamespace, "-o", "jsonpath={.status.phase}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("ManifestGenerationInProgress"))
		})

		It("should complete Phase 2: ArgoCD Application creation after PipelineRun succeeds", func() {
			By("Creating HeliosApp")
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(heliosAppYAML)
			_, _ = utils.Run(cmd)

			By("Waiting for PipelineRun to be created")
			var pipelineRunName string
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", testNamespace, "-o", "jsonpath={.items[0].metadata.name}")
				output, _ := utils.Run(cmd)
				pipelineRunName = output
				return output
			}, timeout, interval).ShouldNot(BeEmpty())

			By("Simulating PipelineRun success by patching status")
			patchJSON := `{"status":{"conditions":[{"type":"Succeeded","status":"True","reason":"Succeeded"}]}}`
			cmd = exec.Command("kubectl", "patch", "pipelinerun", pipelineRunName, "-n", testNamespace,
				"--type=merge", "--subresource=status", "-p", patchJSON)
			_, _ = utils.Run(cmd)

			By("Waiting for ArgoCD Application to be created")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "application", "-n", "argocd", "-o", "name")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(ContainSubstring(testAppName))

			By("Checking HeliosApp status shows Phase 2")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "heliosapp", testAppName, "-n", testNamespace, "-o", "jsonpath={.status.phase}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("DeployingWithArgoCD"))
		})

		It("should complete Phase 3: Status sync from ArgoCD", func() {
			By("Creating HeliosApp and simulating full flow")
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(heliosAppYAML)
			_, _ = utils.Run(cmd)

			By("Waiting for PipelineRun and patching to success")
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", testNamespace, "-o", "jsonpath={.items[0].metadata.name}")
				output, err := utils.Run(cmd)
				if err != nil || output == "" {
					return fmt.Errorf("no pipelinerun found")
				}

				patchJSON := `{"status":{"conditions":[{"type":"Succeeded","status":"True","reason":"Succeeded"}]}}`
				cmd = exec.Command("kubectl", "patch", "pipelinerun", strings.TrimSpace(output), "-n", testNamespace,
					"--type=merge", "--subresource=status", "-p", patchJSON)
				_, err = utils.Run(cmd)
				return err
			}, timeout, interval).Should(Succeed())

			By("Simulating ArgoCD Application status")
			// In real scenario, ArgoCD would update this
			// For E2E, we verify the operator creates the Application
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", testAppName, "-n", "argocd")
				_, err := utils.Run(cmd)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Verifying observedGeneration is updated")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "heliosapp", testAppName, "-n", testNamespace,
					"-o", "jsonpath={.status.observedGeneration}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("1"))
		})

		It("should handle spec changes and trigger new PipelineRun", func() {
			By("Creating initial HeliosApp")
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(heliosAppYAML)
			_, _ = utils.Run(cmd)

			By("Waiting for first PipelineRun")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", testNamespace, "-o", "name")
				output, _ := utils.Run(cmd)
				return len(strings.Split(strings.TrimSpace(output), "\n"))
			}, timeout, interval).Should(BeNumerically(">=", 1))

			initialCount := func() int {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", testNamespace, "-o", "name")
				output, _ := utils.Run(cmd)
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if lines[0] == "" {
					return 0
				}
				return len(lines)
			}()

			By("Updating HeliosApp spec")
			patchJSON := `{"spec":{"replicas":5}}`
			cmd = exec.Command("kubectl", "patch", "heliosapp", testAppName, "-n", testNamespace,
				"--type=merge", "-p", patchJSON)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying new PipelineRun is created")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", testNamespace, "-o", "name")
				output, _ := utils.Run(cmd)
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if lines[0] == "" {
					return 0
				}
				return len(lines)
			}, timeout, interval).Should(BeNumerically(">", initialCount))

			By("Verifying generation incremented")
			cmd = exec.Command("kubectl", "get", "heliosapp", testAppName, "-n", testNamespace,
				"-o", "jsonpath={.metadata.generation}")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("2"))
		})

		It("should handle PipelineRun failures gracefully", func() {
			By("Creating HeliosApp")
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(heliosAppYAML)
			_, _ = utils.Run(cmd)

			By("Waiting for PipelineRun")
			var pipelineRunName string
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", testNamespace, "-o", "jsonpath={.items[0].metadata.name}")
				output, _ := utils.Run(cmd)
				pipelineRunName = strings.TrimSpace(output)
				return pipelineRunName
			}, timeout, interval).ShouldNot(BeEmpty())

			By("Simulating PipelineRun failure")
			patchJSON := `{"status":{"conditions":[{"type":"Succeeded","status":"False","reason":"Failed","message":"Mock failure"}]}}`
			cmd = exec.Command("kubectl", "patch", "pipelinerun", pipelineRunName, "-n", testNamespace,
				"--type=merge", "--subresource=status", "-p", patchJSON)
			_, _ = utils.Run(cmd)

			By("Checking HeliosApp status reflects failure")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "heliosapp", testAppName, "-n", testNamespace,
					"-o", "jsonpath={.status.phase}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("ManifestGenerationFailed"))

			By("Verifying failure message in status")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "heliosapp", testAppName, "-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(ContainSubstring("Mock failure"))
		})
	})
})
