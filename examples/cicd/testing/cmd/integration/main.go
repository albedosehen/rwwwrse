package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IntegrationTester manages the integration testing of the CI/CD pipeline
type IntegrationTester struct {
	config       TestConfig
	results      []TestResult
	startTime    time.Time
	githubToken  string
	repoOwner    string
	repoName     string
	httpClient   *http.Client
}

// TestConfig holds configuration for integration tests
type TestConfig struct {
	ProjectRoot     string        `json:"project_root"`
	TestTimeout     time.Duration `json:"test_timeout"`
	HealthCheckURL  string        `json:"health_check_url"`
	SynologyHost    string        `json:"synology_host"`
	SynologyPort    string        `json:"synology_port"`
	TestBranch      string        `json:"test_branch"`
	TestCommitSHA   string        `json:"test_commit_sha"`
	SkipDeployment  bool          `json:"skip_deployment"`
	SkipSynology    bool          `json:"skip_synology"`
}

// TestResult represents the result of a single test
type TestResult struct {
	TestName    string        `json:"test_name"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	Message     string        `json:"message"`
	Details     interface{}   `json:"details,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	HeadBranch  string    `json:"head_branch"`
	HeadSHA     string    `json:"head_sha"`
	HTMLURL     string    `json:"html_url"`
}

// WorkflowRunsResponse represents the GitHub API response for workflow runs
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// NewIntegrationTester creates a new integration tester
func NewIntegrationTester(config TestConfig) *IntegrationTester {
	return &IntegrationTester{
		config:      config,
		results:     make([]TestResult, 0),
		startTime:   time.Now(),
		githubToken: os.Getenv("GITHUB_TOKEN"),
		repoOwner:   os.Getenv("GITHUB_REPOSITORY_OWNER"),
		repoName:    os.Getenv("GITHUB_REPOSITORY_NAME"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddResult adds a test result
func (it *IntegrationTester) AddResult(testName, status, message string, details interface{}) {
	result := TestResult{
		TestName:  testName,
		Status:    status,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Duration:  time.Since(it.startTime),
	}
	it.results = append(it.results, result)
	
	// Log the result
	statusIcon := "✅"
	if status == "FAILED" {
		statusIcon = "❌"
	} else if status == "WARNING" {
		statusIcon = "⚠️"
	}
	
	fmt.Printf("%s [%s] %s: %s\n", statusIcon, status, testName, message)
}

// TestWorkflowSyntax tests the syntax of all workflow files
func (it *IntegrationTester) TestWorkflowSyntax(ctx context.Context) error {
	fmt.Println("🔍 Testing workflow syntax...")
	
	workflowsDir := filepath.Join(it.config.ProjectRoot, ".github", "workflows")
	
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		it.AddResult("workflow_syntax", "FAILED", "Workflows directory not found", nil)
		return err
	}
	
	var syntaxErrors []string
	
	err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && (strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")) {
			if err := it.validateWorkflowFile(ctx, path); err != nil {
				syntaxErrors = append(syntaxErrors, fmt.Sprintf("%s: %v", path, err))
			}
		}
		
		return nil
	})
	
	if err != nil {
		it.AddResult("workflow_syntax", "FAILED", "Error walking workflows directory", err)
		return err
	}
	
	if len(syntaxErrors) > 0 {
		it.AddResult("workflow_syntax", "FAILED", "Syntax errors found", syntaxErrors)
		return fmt.Errorf("syntax errors found: %v", syntaxErrors)
	}
	
	it.AddResult("workflow_syntax", "PASSED", "All workflow files have valid syntax", nil)
	return nil
}

// validateWorkflowFile validates a single workflow file
func (it *IntegrationTester) validateWorkflowFile(ctx context.Context, filePath string) error {
	// Use yq to validate YAML syntax if available
	if _, err := exec.LookPath("yq"); err == nil {
		cmd := exec.CommandContext(ctx, "yq", "eval", ".", filePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("invalid YAML syntax: %v", err)
		}
	}
	
	// Additional validation can be added here
	return nil
}

// TestCIPipeline tests the CI pipeline execution
func (it *IntegrationTester) TestCIPipeline(ctx context.Context) error {
	fmt.Println("🔨 Testing CI pipeline...")
	
	if it.githubToken == "" {
		it.AddResult("ci_pipeline", "SKIPPED", "GitHub token not available", nil)
		return nil
	}
	
	// Trigger CI workflow or check recent runs
	workflowRuns, err := it.getWorkflowRuns(ctx, "ci.yml")
	if err != nil {
		it.AddResult("ci_pipeline", "FAILED", "Failed to get CI workflow runs", err)
		return err
	}
	
	if len(workflowRuns) == 0 {
		it.AddResult("ci_pipeline", "WARNING", "No CI workflow runs found", nil)
		return nil
	}
	
	// Check the most recent run
	latestRun := workflowRuns[0]
	
	if latestRun.Status == "completed" {
		if latestRun.Conclusion == "success" {
			it.AddResult("ci_pipeline", "PASSED", "Latest CI run completed successfully", latestRun)
		} else {
			it.AddResult("ci_pipeline", "FAILED", fmt.Sprintf("Latest CI run failed: %s", latestRun.Conclusion), latestRun)
			return fmt.Errorf("CI pipeline failed")
		}
	} else {
		it.AddResult("ci_pipeline", "WARNING", fmt.Sprintf("Latest CI run is %s", latestRun.Status), latestRun)
	}
	
	return nil
}

// TestCDPipeline tests the CD pipeline execution
func (it *IntegrationTester) TestCDPipeline(ctx context.Context) error {
	fmt.Println("🚀 Testing CD pipeline...")
	
	if it.config.SkipDeployment {
		it.AddResult("cd_pipeline", "SKIPPED", "Deployment testing skipped", nil)
		return nil
	}
	
	if it.githubToken == "" {
		it.AddResult("cd_pipeline", "SKIPPED", "GitHub token not available", nil)
		return nil
	}
	
	// Check CD workflow runs
	workflowRuns, err := it.getWorkflowRuns(ctx, "cd.yml")
	if err != nil {
		it.AddResult("cd_pipeline", "FAILED", "Failed to get CD workflow runs", err)
		return err
	}
	
	if len(workflowRuns) == 0 {
		it.AddResult("cd_pipeline", "WARNING", "No CD workflow runs found", nil)
		return nil
	}
	
	// Check the most recent run
	latestRun := workflowRuns[0]
	
	if latestRun.Status == "completed" {
		if latestRun.Conclusion == "success" {
			it.AddResult("cd_pipeline", "PASSED", "Latest CD run completed successfully", latestRun)
		} else {
			it.AddResult("cd_pipeline", "FAILED", fmt.Sprintf("Latest CD run failed: %s", latestRun.Conclusion), latestRun)
			return fmt.Errorf("CD pipeline failed")
		}
	} else {
		it.AddResult("cd_pipeline", "WARNING", fmt.Sprintf("Latest CD run is %s", latestRun.Status), latestRun)
	}
	
	return nil
}

// TestSynologyDeployment tests the Synology deployment
func (it *IntegrationTester) TestSynologyDeployment(ctx context.Context) error {
	fmt.Println("🏠 Testing Synology deployment...")
	
	if it.config.SkipSynology {
		it.AddResult("synology_deployment", "SKIPPED", "Synology testing skipped", nil)
		return nil
	}
	
	// Test Synology workflow runs
	if it.githubToken != "" {
		workflowRuns, err := it.getWorkflowRuns(ctx, "syno.yaml")
		if err != nil {
			it.AddResult("synology_deployment", "WARNING", "Failed to get Synology workflow runs", err)
		} else if len(workflowRuns) > 0 {
			latestRun := workflowRuns[0]
			if latestRun.Status == "completed" && latestRun.Conclusion == "success" {
				it.AddResult("synology_deployment", "PASSED", "Latest Synology deployment successful", latestRun)
			} else {
				it.AddResult("synology_deployment", "WARNING", fmt.Sprintf("Latest Synology run: %s/%s", latestRun.Status, latestRun.Conclusion), latestRun)
			}
		}
	}
	
	// Test Synology health endpoint if available
	if it.config.SynologyHost != "" && it.config.SynologyPort != "" {
		healthURL := fmt.Sprintf("http://%s:%s/health", it.config.SynologyHost, it.config.SynologyPort)
		if err := it.testHealthEndpoint(ctx, healthURL, "synology"); err != nil {
			it.AddResult("synology_health", "FAILED", "Synology health check failed", err)
			return err
		}
		it.AddResult("synology_health", "PASSED", "Synology health check successful", nil)
	} else {
		it.AddResult("synology_health", "SKIPPED", "Synology host/port not configured", nil)
	}
	
	return nil
}

// testHealthEndpoint tests a health endpoint
func (it *IntegrationTester) testHealthEndpoint(ctx context.Context, url, service string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := it.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// getWorkflowRuns gets workflow runs from GitHub API
func (it *IntegrationTester) getWorkflowRuns(ctx context.Context, workflowFile string) ([]WorkflowRun, error) {
	if it.repoOwner == "" || it.repoName == "" {
		return nil, fmt.Errorf("repository information not available")
	}
	
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/runs", it.repoOwner, it.repoName, workflowFile)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "token "+it.githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	resp, err := it.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API request failed with status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var response WorkflowRunsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	return response.WorkflowRuns, nil
}

// GenerateReport generates a comprehensive test report
func (it *IntegrationTester) GenerateReport() IntegrationTestReport {
	totalTests := len(it.results)
	passedTests := 0
	failedTests := 0
	warningTests := 0
	skippedTests := 0
	
	for _, result := range it.results {
		switch result.Status {
		case "PASSED":
			passedTests++
		case "FAILED":
			failedTests++
		case "WARNING":
			warningTests++
		case "SKIPPED":
			skippedTests++
		}
	}
	
	overallStatus := "PASSED"
	if failedTests > 0 {
		overallStatus = "FAILED"
	} else if warningTests > 0 {
		overallStatus = "WARNING"
	}
	
	return IntegrationTestReport{
		Timestamp: time.Now(),
		Duration:  time.Since(it.startTime),
		Summary: TestSummary{
			TotalTests:    totalTests,
			PassedTests:   passedTests,
			FailedTests:   failedTests,
			WarningTests:  warningTests,
			SkippedTests:  skippedTests,
			OverallStatus: overallStatus,
		},
		Results: it.results,
		Config:  it.config,
	}
}

// IntegrationTestReport represents the complete integration test report
type IntegrationTestReport struct {
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	Summary   TestSummary   `json:"summary"`
	Results   []TestResult  `json:"results"`
	Config    TestConfig    `json:"config"`
}

// TestSummary provides a summary of test results
type TestSummary struct {
	TotalTests    int    `json:"total_tests"`
	PassedTests   int    `json:"passed_tests"`
	FailedTests   int    `json:"failed_tests"`
	WarningTests  int    `json:"warning_tests"`
	SkippedTests  int    `json:"skipped_tests"`
	OverallStatus string `json:"overall_status"`
}

// RunIntegrationTests runs all integration tests
func (it *IntegrationTester) RunIntegrationTests(ctx context.Context) error {
	fmt.Println("🧪 Starting CI/CD Integration Tests")
	fmt.Println("===================================")
	
	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Workflow Syntax", it.TestWorkflowSyntax},
		{"CI Pipeline", it.TestCIPipeline},
		{"CD Pipeline", it.TestCDPipeline},
		{"Synology Deployment", it.TestSynologyDeployment},
	}
	
	for _, test := range tests {
		fmt.Printf("\n🔍 Running: %s\n", test.name)
		if err := test.fn(ctx); err != nil {
			fmt.Printf("❌ Test failed: %s - %v\n", test.name, err)
		}
	}
	
	return nil
}

func main() {
	ctx := context.Background()
	
	// Load configuration
	config := TestConfig{
		ProjectRoot:     ".",
		TestTimeout:     30 * time.Minute,
		HealthCheckURL:  "http://localhost:8080/health",
		SynologyHost:    os.Getenv("SYNOLOGY_HOST"),
		SynologyPort:    os.Getenv("SYNOLOGY_PORT"),
		TestBranch:      os.Getenv("GITHUB_REF_NAME"),
		TestCommitSHA:   os.Getenv("GITHUB_SHA"),
		SkipDeployment:  os.Getenv("SKIP_DEPLOYMENT") == "true",
		SkipSynology:    os.Getenv("SKIP_SYNOLOGY") == "true",
	}
	
	// Create tester
	tester := NewIntegrationTester(config)
	
	// Run tests
	if err := tester.RunIntegrationTests(ctx); err != nil {
		log.Printf("Integration tests encountered errors: %v", err)
	}
	
	// Generate report
	report := tester.GenerateReport()
	
	// Print summary
	fmt.Println("\n📊 Integration Test Summary")
	fmt.Println("===========================")
	fmt.Printf("Overall Status: %s\n", report.Summary.OverallStatus)
	fmt.Printf("Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Printf("Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("Failed: %d\n", report.Summary.FailedTests)
	fmt.Printf("Warnings: %d\n", report.Summary.WarningTests)
	fmt.Printf("Skipped: %d\n", report.Summary.SkippedTests)
	fmt.Printf("Duration: %v\n", report.Duration)
	
	// Save report
	reportFile := "integration-test-report.json"
	if reportData, err := json.MarshalIndent(report, "", "  "); err == nil {
		if err := os.WriteFile(reportFile, reportData, 0644); err == nil {
			fmt.Printf("\n📄 Report saved to: %s\n", reportFile)
		}
	}
	
	// Exit with appropriate code
	if report.Summary.OverallStatus == "FAILED" {
		os.Exit(1)
	}
}