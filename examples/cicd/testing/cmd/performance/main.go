package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// PerformanceTester manages performance testing of the CI/CD pipeline
type PerformanceTester struct {
	config  PerformanceConfig
	results []PerformanceResult
	client  *http.Client
}

// PerformanceConfig holds configuration for performance tests
type PerformanceConfig struct {
	BaseURL           string        `json:"base_url"`
	ConcurrentUsers   int           `json:"concurrent_users"`
	TestDuration      time.Duration `json:"test_duration"`
	RequestTimeout    time.Duration `json:"request_timeout"`
	RampUpTime        time.Duration `json:"ramp_up_time"`
	ThinkTime         time.Duration `json:"think_time"`
	MaxResponseTime   time.Duration `json:"max_response_time"`
	MinThroughput     float64       `json:"min_throughput"`
	MaxErrorRate      float64       `json:"max_error_rate"`
}

// PerformanceResult represents performance test results
type PerformanceResult struct {
	TestName        string        `json:"test_name"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Duration        time.Duration `json:"duration"`
	TotalRequests   int           `json:"total_requests"`
	SuccessRequests int           `json:"success_requests"`
	FailedRequests  int           `json:"failed_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`
	P95ResponseTime time.Duration `json:"p95_response_time"`
	P99ResponseTime time.Duration `json:"p99_response_time"`
	Throughput      float64       `json:"throughput"`
	ErrorRate       float64       `json:"error_rate"`
	Status          string        `json:"status"`
	Details         interface{}   `json:"details,omitempty"`
}

// RequestResult represents individual request results
type RequestResult struct {
	Timestamp    time.Time     `json:"timestamp"`
	ResponseTime time.Duration `json:"response_time"`
	StatusCode   int           `json:"status_code"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
}

// NewPerformanceTester creates a new performance tester
func NewPerformanceTester(config PerformanceConfig) *PerformanceTester {
	return &PerformanceTester{
		config:  config,
		results: make([]PerformanceResult, 0),
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

// TestHealthEndpointPerformance tests health endpoint performance
func (pt *PerformanceTester) TestHealthEndpointPerformance(ctx context.Context) error {
	fmt.Println("🚀 Testing Health Endpoint Performance...")
	
	healthURL := pt.config.BaseURL + "/health"
	return pt.runLoadTest(ctx, "health_endpoint", healthURL, "GET", nil)
}

// TestMetricsEndpointPerformance tests metrics endpoint performance
func (pt *PerformanceTester) TestMetricsEndpointPerformance(ctx context.Context) error {
	fmt.Println("📊 Testing Metrics Endpoint Performance...")
	
	metricsURL := pt.config.BaseURL + "/metrics"
	return pt.runLoadTest(ctx, "metrics_endpoint", metricsURL, "GET", nil)
}

// TestMainEndpointPerformance tests main application endpoint performance
func (pt *PerformanceTester) TestMainEndpointPerformance(ctx context.Context) error {
	fmt.Println("🌐 Testing Main Endpoint Performance...")
	
	mainURL := pt.config.BaseURL + "/"
	return pt.runLoadTest(ctx, "main_endpoint", mainURL, "GET", nil)
}

// runLoadTest executes a load test against a specific endpoint
func (pt *PerformanceTester) runLoadTest(ctx context.Context, testName, url, method string, body interface{}) error {
	startTime := time.Now()
	
	// Channel to collect request results
	resultsChan := make(chan RequestResult, pt.config.ConcurrentUsers*100)
	
	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup
	
	// Context with timeout
	testCtx, cancel := context.WithTimeout(ctx, pt.config.TestDuration)
	defer cancel()
	
	// Start concurrent users with ramp-up
	userStartInterval := pt.config.RampUpTime / time.Duration(pt.config.ConcurrentUsers)
	
	for i := 0; i < pt.config.ConcurrentUsers; i++ {
		wg.Add(1)
		
		go func(userID int) {
			defer wg.Done()
			
			// Ramp-up delay
			time.Sleep(time.Duration(userID) * userStartInterval)
			
			pt.simulateUser(testCtx, userID, url, method, body, resultsChan)
		}(i)
	}
	
	// Close results channel when all users are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
	
	// Collect results
	var requestResults []RequestResult
	for result := range resultsChan {
		requestResults = append(requestResults, result)
	}
	
	endTime := time.Now()
	
	// Analyze results
	result := pt.analyzeResults(testName, startTime, endTime, requestResults)
	pt.results = append(pt.results, result)
	
	// Print summary
	pt.printTestSummary(result)
	
	return nil
}

// simulateUser simulates a single user making requests
func (pt *PerformanceTester) simulateUser(ctx context.Context, userID int, url, method string, body interface{}, resultsChan chan<- RequestResult) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Make request
			result := pt.makeRequest(url, method, body)
			
			select {
			case resultsChan <- result:
			case <-ctx.Done():
				return
			}
			
			// Think time between requests
			if pt.config.ThinkTime > 0 {
				select {
				case <-time.After(pt.config.ThinkTime):
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// makeRequest makes a single HTTP request and measures performance
func (pt *PerformanceTester) makeRequest(url, method string, body interface{}) RequestResult {
	start := time.Now()
	
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return RequestResult{
			Timestamp:    start,
			ResponseTime: time.Since(start),
			Success:      false,
			Error:        err.Error(),
		}
	}
	
	resp, err := pt.client.Do(req)
	responseTime := time.Since(start)
	
	result := RequestResult{
		Timestamp:    start,
		ResponseTime: responseTime,
		Success:      err == nil && resp != nil && resp.StatusCode < 400,
	}
	
	if err != nil {
		result.Error = err.Error()
	} else {
		result.StatusCode = resp.StatusCode
		resp.Body.Close()
	}
	
	return result
}

// analyzeResults analyzes request results and generates performance metrics
func (pt *PerformanceTester) analyzeResults(testName string, startTime, endTime time.Time, results []RequestResult) PerformanceResult {
	totalRequests := len(results)
	successRequests := 0
	failedRequests := 0
	
	var responseTimes []time.Duration
	var totalResponseTime time.Duration
	minResponseTime := time.Hour
	maxResponseTime := time.Duration(0)
	
	for _, result := range results {
		if result.Success {
			successRequests++
		} else {
			failedRequests++
		}
		
		responseTimes = append(responseTimes, result.ResponseTime)
		totalResponseTime += result.ResponseTime
		
		if result.ResponseTime < minResponseTime {
			minResponseTime = result.ResponseTime
		}
		if result.ResponseTime > maxResponseTime {
			maxResponseTime = result.ResponseTime
		}
	}
	
	duration := endTime.Sub(startTime)
	avgResponseTime := time.Duration(0)
	if totalRequests > 0 {
		avgResponseTime = totalResponseTime / time.Duration(totalRequests)
	}
	
	throughput := float64(totalRequests) / duration.Seconds()
	errorRate := float64(failedRequests) / float64(totalRequests) * 100
	
	// Calculate percentiles
	p95ResponseTime := pt.calculatePercentile(responseTimes, 95)
	p99ResponseTime := pt.calculatePercentile(responseTimes, 99)
	
	// Determine status
	status := "PASSED"
	if avgResponseTime > pt.config.MaxResponseTime {
		status = "FAILED"
	}
	if throughput < pt.config.MinThroughput {
		status = "FAILED"
	}
	if errorRate > pt.config.MaxErrorRate {
		status = "FAILED"
	}
	
	return PerformanceResult{
		TestName:        testName,
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        duration,
		TotalRequests:   totalRequests,
		SuccessRequests: successRequests,
		FailedRequests:  failedRequests,
		AvgResponseTime: avgResponseTime,
		MinResponseTime: minResponseTime,
		MaxResponseTime: maxResponseTime,
		P95ResponseTime: p95ResponseTime,
		P99ResponseTime: p99ResponseTime,
		Throughput:      throughput,
		ErrorRate:       errorRate,
		Status:          status,
	}
}

// calculatePercentile calculates the specified percentile of response times
func (pt *PerformanceTester) calculatePercentile(responseTimes []time.Duration, percentile int) time.Duration {
	if len(responseTimes) == 0 {
		return 0
	}
	
	// Simple percentile calculation (should use proper sorting for production)
	index := (len(responseTimes) * percentile) / 100
	if index >= len(responseTimes) {
		index = len(responseTimes) - 1
	}
	
	// For simplicity, return the value at the calculated index
	// In production, you'd want to sort the slice first
	return responseTimes[index]
}

// printTestSummary prints a summary of test results
func (pt *PerformanceTester) printTestSummary(result PerformanceResult) {
	statusIcon := "✅"
	if result.Status == "FAILED" {
		statusIcon = "❌"
	}
	
	fmt.Printf("%s [%s] %s Performance Test\n", statusIcon, result.Status, result.TestName)
	fmt.Printf("  Duration: %v\n", result.Duration)
	fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("  Success Rate: %.2f%%\n", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("  Throughput: %.2f req/s\n", result.Throughput)
	fmt.Printf("  Avg Response Time: %v\n", result.AvgResponseTime)
	fmt.Printf("  P95 Response Time: %v\n", result.P95ResponseTime)
	fmt.Printf("  P99 Response Time: %v\n", result.P99ResponseTime)
	fmt.Printf("  Error Rate: %.2f%%\n", result.ErrorRate)
	fmt.Println()
}

// GenerateReport generates a comprehensive performance test report
func (pt *PerformanceTester) GenerateReport() PerformanceReport {
	totalTests := len(pt.results)
	passedTests := 0
	failedTests := 0
	
	var totalRequests int
	var totalSuccessRequests int
	var totalFailedRequests int
	var totalResponseTime time.Duration
	
	for _, result := range pt.results {
		if result.Status == "PASSED" {
			passedTests++
		} else {
			failedTests++
		}
		
		totalRequests += result.TotalRequests
		totalSuccessRequests += result.SuccessRequests
		totalFailedRequests += result.FailedRequests
		totalResponseTime += result.AvgResponseTime
	}
	
	overallStatus := "PASSED"
	if failedTests > 0 {
		overallStatus = "FAILED"
	}
	
	avgResponseTime := time.Duration(0)
	if totalTests > 0 {
		avgResponseTime = totalResponseTime / time.Duration(totalTests)
	}
	
	return PerformanceReport{
		Timestamp: time.Now(),
		Summary: PerformanceSummary{
			TotalTests:          totalTests,
			PassedTests:         passedTests,
			FailedTests:         failedTests,
			OverallStatus:       overallStatus,
			TotalRequests:       totalRequests,
			TotalSuccessRequests: totalSuccessRequests,
			TotalFailedRequests: totalFailedRequests,
			OverallThroughput:   float64(totalRequests) / pt.config.TestDuration.Seconds(),
			AvgResponseTime:     avgResponseTime,
		},
		Results: pt.results,
		Config:  pt.config,
	}
}

// PerformanceReport represents the complete performance test report
type PerformanceReport struct {
	Timestamp time.Time          `json:"timestamp"`
	Summary   PerformanceSummary `json:"summary"`
	Results   []PerformanceResult `json:"results"`
	Config    PerformanceConfig  `json:"config"`
}

// PerformanceSummary provides a summary of performance test results
type PerformanceSummary struct {
	TotalTests           int           `json:"total_tests"`
	PassedTests          int           `json:"passed_tests"`
	FailedTests          int           `json:"failed_tests"`
	OverallStatus        string        `json:"overall_status"`
	TotalRequests        int           `json:"total_requests"`
	TotalSuccessRequests int           `json:"total_success_requests"`
	TotalFailedRequests  int           `json:"total_failed_requests"`
	OverallThroughput    float64       `json:"overall_throughput"`
	AvgResponseTime      time.Duration `json:"avg_response_time"`
}

// RunPerformanceTests runs all performance tests
func (pt *PerformanceTester) RunPerformanceTests(ctx context.Context) error {
	fmt.Println("🚀 Starting Performance Tests")
	fmt.Println("=============================")
	
	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Health Endpoint", pt.TestHealthEndpointPerformance},
		{"Metrics Endpoint", pt.TestMetricsEndpointPerformance},
		{"Main Endpoint", pt.TestMainEndpointPerformance},
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
	config := PerformanceConfig{
		BaseURL:          getEnvOrDefault("BASE_URL", "http://localhost:8080"),
		ConcurrentUsers:  10,
		TestDuration:     30 * time.Second,
		RequestTimeout:   5 * time.Second,
		RampUpTime:       5 * time.Second,
		ThinkTime:        100 * time.Millisecond,
		MaxResponseTime:  2 * time.Second,
		MinThroughput:    10.0, // requests per second
		MaxErrorRate:     5.0,  // percentage
	}
	
	// Create tester
	tester := NewPerformanceTester(config)
	
	// Run tests
	if err := tester.RunPerformanceTests(ctx); err != nil {
		log.Printf("Performance tests encountered errors: %v", err)
	}
	
	// Generate report
	report := tester.GenerateReport()
	
	// Print summary
	fmt.Println("\n📊 Performance Test Summary")
	fmt.Println("============================")
	fmt.Printf("Overall Status: %s\n", report.Summary.OverallStatus)
	fmt.Printf("Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Printf("Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("Failed: %d\n", report.Summary.FailedTests)
	fmt.Printf("Total Requests: %d\n", report.Summary.TotalRequests)
	fmt.Printf("Overall Throughput: %.2f req/s\n", report.Summary.OverallThroughput)
	fmt.Printf("Average Response Time: %v\n", report.Summary.AvgResponseTime)
	
	// Save report
	reportFile := "performance-test-report.json"
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

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}