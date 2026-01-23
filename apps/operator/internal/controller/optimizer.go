package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	heliosappv1 "github.com/hoangphuc841/helios-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OptimizerService handles the logic for detecting waste and auto-fixing it in Git
type OptimizerService struct{}

// OptimizationResult holds the analysis result
type OptimizationResult struct {
	IsWasteful      bool
	CurrentCpu      string
	SuggestedCpu    string
	WastePercentage int
	Strategy        string // "Prometheus P99" or "Instant Heuristic"
	MeasuredCpu     string // Actual usage measured from cluster
	MeasuredMemory  string // Actual usage measured from cluster
}

// Prometheus Query Constants
const (
	PrometheusUrl = "http://prometheus-server.monitoring.svc.cluster.local:9090"                                                                     // Default in-cluster URL
	QueryCpuP99   = "quantile_over_time(0.99, rate(container_cpu_usage_seconds_total{pod=\"%s\", namespace=\"%s\", container!=\"POD\"}[5m])[7d:5m])" // 7-day P99
	QueryMemMax   = "max_over_time(container_memory_usage_bytes{pod=\"%s\", namespace=\"%s\"}[7d])"                                                  // 7-day Max for OOM safety
)

// AnalyzeResourceUsage acts as a Facade: Tries Prometheus first, then falls back to Heuristic
func (s *OptimizerService) AnalyzeResourceUsage(ctx context.Context, k8sClient client.Client, app *heliosappv1.HeliosApp) OptimizationResult {
	logger := log.FromContext(ctx)

	// STRATEGY 1: PROMETHEUS (Production Grade - Chapter 4 Logic)
	// Try to reach Prometheus first
	promResult, err := s.AnalyzePrometheus(ctx, app)
	if err == nil {
		logger.Info("✅ Optimized using PROMETHEUS P99 Strategy", "app", app.Name, "cpu", promResult.SuggestedCpu)
		return promResult
	} else {
		// Log the failure but continue to fallback
		logger.Info("⚠️ Prometheus unreachable, falling back to Instant Heuristic", "error", err)
	}

	// STRATEGY 2: INSTANT HEURISTIC (Fallback / Demo Mode)
	return s.AnalyzeHeuristic(ctx, k8sClient, app)
}

// AnalyzePrometheus implements the "Chapter 4" Logic: 7-day Lookback, P99, OOM Safety
func (s *OptimizerService) AnalyzePrometheus(ctx context.Context, app *heliosappv1.HeliosApp) (OptimizationResult, error) {
	logger := log.FromContext(ctx)

	if app.Spec.Resources == nil || app.Spec.Resources.Requests == nil {
		return OptimizationResult{IsWasteful: false}, fmt.Errorf("no resource requests defined")
	}

	requestCpuStr := app.Spec.Resources.Requests["cpu"]
	requestCpuMillis := parseCpu(requestCpuStr)

	// Get pod name pattern for Prometheus query
	podSelector := fmt.Sprintf("%s-.*", app.Name)

	// Query 1: CPU P99 over 7 days
	cpuQuery := fmt.Sprintf(
		`quantile_over_time(0.99, sum(rate(container_cpu_usage_seconds_total{pod=~"%s", namespace="%s", container!="POD", container!=""}[5m]))[7d:5m])`,
		podSelector, app.Namespace,
	)

	cpuP99, err := s.queryPrometheus(ctx, cpuQuery)
	if err != nil {
		return OptimizationResult{}, fmt.Errorf("prometheus CPU query failed: %v", err)
	}

	// Query 2: Memory Max over 7 days (for OOM safety)
	memQuery := fmt.Sprintf(
		`max_over_time(sum(container_memory_working_set_bytes{pod=~"%s", namespace="%s", container!="POD", container!=""})[7d:5m])`,
		podSelector, app.Namespace,
	)

	memMax, err := s.queryPrometheus(ctx, memQuery)
	if err != nil {
		logger.Info("Memory query failed, continuing with CPU only", "error", err)
		memMax = 0
	}

	// Convert CPU from cores to millicores
	cpuP99Millis := int64(cpuP99 * 1000)

	// Convert Memory from bytes to Mi
	memMaxMi := int64(memMax / (1024 * 1024))

	logger.Info("Prometheus metrics retrieved",
		"app", app.Name,
		"cpuP99", fmt.Sprintf("%dm", cpuP99Millis),
		"memMax", fmt.Sprintf("%dMi", memMaxMi),
		"currentRequest", requestCpuStr,
	)

	// Apply Safety Guardrails
	// Rule 1: CPU buffer 1.2x (compressible, throttling OK)
	suggestedCpuMillis := int64(float64(cpuP99Millis) * 1.2)
	if suggestedCpuMillis < 50 {
		suggestedCpuMillis = 50 // Minimum floor
	}

	// Rule 2: Memory buffer 1.5x (not compressible, OOM is fatal)
	suggestedMemMi := int64(float64(memMaxMi) * 1.5)
	if suggestedMemMi < 128 {
		suggestedMemMi = 128 // Minimum floor
	}

	// Check if optimization is needed (usage < 50% of request)
	threshold := requestCpuMillis / 2
	isWasteful := cpuP99Millis < threshold

	result := OptimizationResult{
		IsWasteful:     isWasteful,
		CurrentCpu:     requestCpuStr,
		SuggestedCpu:   fmt.Sprintf("%dm", suggestedCpuMillis),
		MeasuredCpu:    fmt.Sprintf("%dm", cpuP99Millis),
		MeasuredMemory: fmt.Sprintf("%dMi", memMaxMi),
		Strategy:       "Prometheus P99 (7-day)",
	}

	if isWasteful {
		result.WastePercentage = int((float64(requestCpuMillis-cpuP99Millis) / float64(requestCpuMillis)) * 100)
		logger.Info("Waste detected via Prometheus P99",
			"waste%", result.WastePercentage,
			"suggested", result.SuggestedCpu,
		)
	}

	return result, nil
}

// queryPrometheus makes HTTP request to Prometheus API and returns the scalar value
func (s *OptimizerService) queryPrometheus(ctx context.Context, query string) (float64, error) {
	logger := log.FromContext(ctx)

	// Build request URL
	promUrl := os.Getenv("PROMETHEUS_URL")
	if promUrl == "" {
		promUrl = PrometheusUrl // Default in-cluster URL
	}

	reqUrl := fmt.Sprintf("%s/api/v1/query?query=%s", promUrl, url.QueryEscape(query))

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("prometheus request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Parse Prometheus response JSON
	var promResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"` // [timestamp, "value"]
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &promResp); err != nil {
		return 0, fmt.Errorf("failed to parse prometheus response: %v", err)
	}

	if promResp.Status != "success" {
		return 0, fmt.Errorf("prometheus query failed: %s", promResp.Status)
	}

	if len(promResp.Data.Result) == 0 {
		return 0, fmt.Errorf("no data returned from prometheus")
	}

	// Extract value (second element in Value array)
	valueStr, ok := promResp.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected value format from prometheus")
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse value: %v", err)
	}

	logger.V(1).Info("Prometheus query result", "query", query[:50], "value", value)

	return value, nil
}

// AnalyzeHeuristic utilizes real PodMetrics from K8s API (Instantaneous)
func (s *OptimizerService) AnalyzeHeuristic(ctx context.Context, k8sClient client.Client, app *heliosappv1.HeliosApp) OptimizationResult {
	if app.Spec.Resources == nil || app.Spec.Resources.Requests == nil {
		return OptimizationResult{IsWasteful: false}
	}

	requestCpuStr := app.Spec.Resources.Requests["cpu"] // e.g., "250m"
	requestCpuMillis := parseCpu(requestCpuStr)

	logger := log.FromContext(ctx)
	logger.Info("Analyzing resource usage (Heuristic Mode)", "app", app.Name, "request", requestCpuStr)

	// 1. Get Pods for this App
	podList := &corev1.PodList{}
	err := k8sClient.List(ctx, podList, client.InNamespace(app.Namespace), client.MatchingLabels{"app": app.Name})

	if err != nil || len(podList.Items) == 0 {
		logger.Info("No pods found for app, skipping optimization", "app", app.Name)
		return OptimizationResult{IsWasteful: false}
	}

	// 2. Get Metrics for the first Pod
	podName := podList.Items[0].Name
	podMetrics := &unstructured.Unstructured{}
	podMetrics.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "metrics.k8s.io",
		Version: "v1beta1",
		Kind:    "PodMetrics",
	})

	err = k8sClient.Get(ctx, client.ObjectKey{Name: podName, Namespace: app.Namespace}, podMetrics)
	if err != nil {
		logger.Info("Metrics not available yet", "pod", podName, "error", err)
		return OptimizationResult{IsWasteful: false}
	}

	// 3. Extract Usage
	containers, found, _ := unstructured.NestedSlice(podMetrics.Object, "containers")
	if !found || len(containers) == 0 {
		return OptimizationResult{IsWasteful: false}
	}

	firstContainer := containers[0].(map[string]interface{})
	usageMap, _, _ := unstructured.NestedMap(firstContainer, "usage")
	usageCpuStr := usageMap["cpu"].(string)
	usageCpuMillis := parseCpu(usageCpuStr)

	// 4. Advanced Logic: "Reactive Heuristic" (Phase 1 Logic - Instant)
	// Safety Rule #1: Memory is critical. OOM is fatal. Use higher buffer (1.5x).
	// Safety Rule #2: CPU is compressible. Throttling is acceptable. Use moderate buffer (1.2x).
	// Safety Rule #3: Never go below Minimum Floor (50m / 128Mi).

	// 4. Advanced Logic: "Reactive Heuristic" (Phase 1 Logic - Instant)
	// Safety Rule #1: Memory is critical. OOM is fatal. Use higher buffer (1.5x).
	// Safety Rule #2: CPU is compressible. Throttling is acceptable. Use moderate buffer (1.2x).
	// Safety Rule #3: Never go below Minimum Floor (50m / 128Mi).

	threshold := requestCpuMillis / 2

	// Always return measured usage
	result := OptimizationResult{
		IsWasteful:     false,
		MeasuredCpu:    usageCpuStr,
		MeasuredMemory: "", // TODO: Add memory extraction if needed, for now CPU is priority
	}

	if usageCpuMillis < threshold {
		logger.Info("Waste detected! Applying Safety Guardrails...", "usage", usageCpuMillis, "threshold", threshold)

		// Calculate Target CPU (1.2x Buffer)
		suggestedCpuMillis := int64(float64(usageCpuMillis) * 1.2)
		if suggestedCpuMillis < 50 {
			suggestedCpuMillis = 50 // Minimum Floor
		}

		suggestedCpu := fmt.Sprintf("%dm", suggestedCpuMillis)

		wastePercent := int((float64(requestCpuMillis-usageCpuMillis) / float64(requestCpuMillis)) * 100)

		result.IsWasteful = true
		result.CurrentCpu = requestCpuStr
		result.SuggestedCpu = suggestedCpu
		result.WastePercentage = wastePercent
		result.Strategy = "Instant Heuristic"
	}

	return result
}

// Helper to parse "100m", "1", "10500n" to millicores (int64)
func parseCpu(val string) int64 {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "m") {
		// "100m" -> 100
		v, _ := strconv.ParseInt(strings.TrimSuffix(val, "m"), 10, 64)
		return v
	} else if strings.HasSuffix(val, "n") {
		// "1000000n" -> 1m. nano = 10^-9, milli = 10^-3. 1m = 1,000,000n
		// So n / 1,000,000 = m
		v, _ := strconv.ParseInt(strings.TrimSuffix(val, "n"), 10, 64)
		return v / 1000000
	} else {
		// "1" -> 1000m
		v, _ := strconv.ParseFloat(val, 64) // Use float for "0.5" cases
		return int64(v * 1000)
	}
}

// AutoFixRepository performs the "Closed-Loop GitOps" action
// It clones the user's repo, edits helios.yaml, and pushes the fix.
func (s *OptimizerService) AutoFixRepository(ctx context.Context, app *heliosappv1.HeliosApp, result OptimizationResult) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Auto-Optimization sequence", "app", app.Name, "waste", result.WastePercentage)

	// 1. Prepare Workspace
	tempDir, err := os.MkdirTemp("", "helios-optimizer-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir) // Cleanup after done

	// 2. Clone Repository
	// NOTE: In a real app, we would use the Secret to get the PAT.
	// For this PoC, we assume the environment/GitRepo URL has auth embedded or public.
	repoUrl := app.Spec.GitRepo
	logger.Info("Cloning repository", "url", repoUrl, "dir", tempDir)

	if err := runGitCmd(tempDir, "clone", repoUrl, "."); err != nil {
		return fmt.Errorf("failed to clone repo: %v", err)
	}

	// 3. Locate and Edit helios.yaml
	// We look for helios.yaml in the root or context subpath
	heliosFile := filepath.Join(tempDir, app.Spec.ContextSubpath, "helios.yaml")

	// Read file
	content, err := os.ReadFile(heliosFile)
	var newContent string

	if os.IsNotExist(err) {
		logger.Info("helios.yaml not found, creating new one with optimized values")
		// Create new file content from the App Spec
		// We use a simple template here for the PoC
		newContent = fmt.Sprintf(`apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: %s
spec:
  gitRepo: %s
  gitBranch: %s
  imageRepo: %s
  port: %d
  replicas: %d
  enableAutoOptimization: true
  resources:
    requests:
      cpu: "%s" # Optimized
      memory: "128Mi"
    limits:
      cpu: "500m" 
      memory: "512Mi"
  templateRepo: "%s"
  templatePath: "%s"
  pipelineName: "%s"
  serviceAccount: "%s"
  webhookSecret: "%s"
  gitopsRepo: "%s"
  gitopsPath: "%s"
  values:
    environment: "dev"
`, app.Name, app.Spec.GitRepo, app.Spec.GitBranch, app.Spec.ImageRepo,
			app.Spec.Port, app.Spec.Replicas, result.SuggestedCpu,
			app.Spec.TemplateRepo, app.Spec.TemplatePath, app.Spec.PipelineName,
			app.Spec.ServiceAccount, app.Spec.WebhookSecret, app.Spec.GitOpsRepo, app.Spec.GitOpsPath)

	} else if err != nil {
		return fmt.Errorf("failed to read helios.yaml: %v", err)
	} else {
		// File exists, regex replace
		contentStr := string(content)
		re := regexp.MustCompile(`cpu:\s*"?` + regexp.QuoteMeta(result.CurrentCpu) + `"?`)
		if !re.MatchString(contentStr) {
			logger.Info("Could not find exact CPU string to replace, skipping", "current", result.CurrentCpu)
			return nil
		}
		newContent = re.ReplaceAllString(contentStr, "cpu: "+result.SuggestedCpu)
	}

	if err := os.WriteFile(heliosFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write fix to helios.yaml: %v", err)
	}

	// 5. Commit to a NEW BRANCH (Not main)
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("helios-optimize-%s", timestamp)
	logger.Info("Creating new branch for PR", "branch", branchName)

	if err := runGitCmd(tempDir, "checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch: %v", err)
	}

	logger.Info("Committing optimization fix")

	// Configure git identity
	runGitCmd(tempDir, "config", "user.email", "bot@helios.io")
	runGitCmd(tempDir, "config", "user.name", "Helios Optimizer Bot")

	if err := runGitCmd(tempDir, "add", "."); err != nil {
		return err
	}

	commitMsg := fmt.Sprintf("chore(finops): rightsizing cpu from %s to %s",
		result.CurrentCpu, result.SuggestedCpu)

	if err := runGitCmd(tempDir, "commit", "-m", commitMsg); err != nil {
		return err
	}

	// Push the new branch
	if err := runGitCmd(tempDir, "push", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push branch: %v", err)
	}

	// 6. Create PR (Using GitHub API or just Logging the URL)
	// Since we might lack a sophisticated GITHUB_TOKEN for API calls in this context,
	// we will output the "Create PR" URL which is a common pattern for CLI tools,
	// or simulated "PR Created" if we assume the Token exists.
	// For this PoC, we validly claim we "Prepare the PR".

	// Construct PR URL for demo purposes
	// repoUrl e.g. https://github.com/user/repo.git -> https://github.com/user/repo/pull/new/branchName
	httpUrl := strings.TrimSuffix(repoUrl, ".git")
	prUrl := fmt.Sprintf("%s/pull/new/%s", httpUrl, branchName)

	logger.Info("Optimization Branch Pushed!", "branch", branchName)
	logger.Info(">>> ACTION REQUIRED: Create PR here: " + prUrl)

	return nil
}

// Helper to run git commands
func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %s - output: %s", args[0], err, string(output))
	}
	return nil
}
