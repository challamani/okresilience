package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/challamani/okresilience/pkg/metrics"
	"github.com/challamani/okresilience/pkg/traffic"
)

// TestCase represents a single resilience test case from the JSON file
type TestCase struct {
	Name                  string      `json:"name"`
	Description           string      `json:"description"`
	PrometheusURL         string      `json:"prometheus_url"`
	ServiceEndpoint       string      `json:"service_endpoint"`
	Namespace             string      `json:"namespace"`
	VirtualService        string      `json:"virtual_service"`
	DestinationRule       string      `json:"destination_rule"`
	NumRequests           int         `json:"num_requests"`
	ExpectedResponseCodes []int       `json:"expected_response_codes"`
	App                   string      `json:"app"`
	Type                  string      `json:"type"`
	Assertions            []Assertion `json:"assertions"`
}

// Assertion represents a metric assertion to validate
type Assertion struct {
	Source     string            `json:"source"`
	MetricName string            `json:"metric_name"`
	Labels     map[string]string `json:"labels"`
	Expected   struct {
		Operator string `json:"operator"`
		Value    int    `json:"value"`
	} `json:"expected"`
}

// TestResult represents the outcome of a test execution
type TestResult struct {
	TestName       string
	TestType       string
	Status         string // "PASSED", "FAILED"
	Message        string
	AssertionCount int
	PassedCount    int
	FailedCount    int
}

const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

var (
	testFilePath   = flag.String("test-file", "resilience-tests.json", "Path to resilience test cases JSON file")
	maxRetries     = flag.Int("max-retries", 5, "Maximum number of retries for metrics validation")
	delaySeconds   = flag.Int("delay-seconds", 5, "Delay in seconds between retry attempts")
	testNameFilter = flag.String("test-name", "", "Optional test name filter to run specific test")
)

func main() {
	flag.Parse()

	// Print welcome banner
	printWelcomeBanner()

	// Load test cases from JSON file
	testCases, err := loadTestCases(*testFilePath)
	if err != nil {
		log.Fatalf("❌ Error loading test cases: %v", err)
	}

	if len(testCases) == 0 {
		log.Fatalf("❌ No test cases found in %s", *testFilePath)
	}

	log.Printf("✓ Loaded %d test case(s) from %s\n", len(testCases), *testFilePath)

	// Execute tests sequentially
	results := []TestResult{}
	for idx, testCase := range testCases {
		// Skip if test name filter is set and doesn't match
		if *testNameFilter != "" && testCase.Name != *testNameFilter {
			continue
		}

		testNum := idx + 1
		log.Printf("\n%s═══════════════════════════════════════════════════════════════%s", colorYellow, colorReset)
		log.Printf("%s[%d/%d] Executing: %s (%s)%s", colorYellow, testNum, len(testCases), testCase.Name, testCase.Type, colorReset)
		log.Printf("%s%s%s", colorYellow, testCase.Description, colorReset)
		log.Printf("%s═══════════════════════════════════════════════════════════════%s\n", colorYellow, colorReset)

		result := executeTest(testCase)
		results = append(results, result)

		// Print result immediately
		printTestResult(result)
	}

	// Print summary
	printTestSummary(results)

	// Exit with failure if any test failed
	for _, result := range results {
		if result.Status == "FAILED" {
			os.Exit(1)
		}
	}
}

// loadTestCases reads and parses the JSON test file
func loadTestCases(filePath string) ([]TestCase, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading test file: %v", err)
	}

	var testCases []TestCase
	if err := json.Unmarshal(data, &testCases); err != nil {
		return nil, fmt.Errorf("error parsing test file: %v", err)
	}

	return testCases, nil
}

// executeTest runs a single test case and validates assertions
func executeTest(testCase TestCase) TestResult {
	result := TestResult{
		TestName:       testCase.Name,
		TestType:       testCase.Type,
		AssertionCount: len(testCase.Assertions),
		PassedCount:    0,
		FailedCount:    0,
	}

	// Validate test case configuration
	if err := validateTestCase(testCase); err != nil {
		result.Status = "FAILED"
		result.Message = fmt.Sprintf("Configuration error: %v", err)
		result.FailedCount = len(testCase.Assertions)
		return result
	}

	// Capture metrics before traffic generation
	beforeMetrics := make(map[int]int) // index -> metric value
	log.Printf("  ⏳ Capturing baseline metrics (%d assertion(s))...", len(testCase.Assertions))
	for idx, assertion := range testCase.Assertions {
		if assertion.Source != "metric" {
			continue
		}

		value, err := queryMetric(testCase.PrometheusURL, assertion.MetricName, assertion.Labels)
		if err != nil {
			log.Printf("  ⚠ Warning: Error querying metric before (assertion %d): %v", idx, err)
			value = 0
		}
		beforeMetrics[idx] = value
		log.Printf("    • Assertion %d: %s = %d", idx, assertion.MetricName, value)
	}
	log.Printf("  ✓ Baseline metrics captured\n")

	// Generate test traffic
	log.Printf("  🚀 Generating %d request(s) to %s...", testCase.NumRequests, testCase.ServiceEndpoint)
	responseCodes, err := traffic.GenerateTraffic(
		testCase.ServiceEndpoint,
		testCase.Namespace,
		testCase.VirtualService,
		testCase.NumRequests,
	)
	if err != nil {
		log.Printf("  ⚠ Warning: Traffic generation had errors: %v", err)
	}

	// Log and validate response codes
	var responseCodesMismatch int
	for i, code := range responseCodes {
		statusIcon := "✓"
		if i < len(testCase.ExpectedResponseCodes) && code != testCase.ExpectedResponseCodes[i] {
			statusIcon = "✗"
			responseCodesMismatch++
		}
		log.Printf("    • Request %d: %s %d", i+1, statusIcon, code)
	}

	// Log response code validation summary
	if len(testCase.ExpectedResponseCodes) > 0 {
		if responseCodesMismatch > 0 {
			log.Printf("  ⚠ Response codes: %d/%d mismatches", responseCodesMismatch, len(responseCodes))
		} else {
			log.Printf("  ✓ All response codes matched expectations")
		}
	}
	log.Printf("  ✓ Traffic generation completed\n")

	// Retry loop for metrics validation
	var afterMetrics map[int]int
	var validationPassed bool

	log.Printf("  ⏳ Validating metrics (max %d retries, %d second delays)...", *maxRetries, *delaySeconds)
	for attempt := 1; attempt <= *maxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("    Retry attempt %d/%d...", attempt, *maxRetries)
		}
		time.Sleep(time.Duration(*delaySeconds) * time.Second)

		afterMetrics = make(map[int]int)
		for idx, assertion := range testCase.Assertions {
			if assertion.Source != "metric" {
				continue
			}

			value, err := queryMetric(testCase.PrometheusURL, assertion.MetricName, assertion.Labels)
			if err != nil {
				log.Printf("    ⚠ Warning: Error querying metric after (assertion %d): %v", idx, err)
				value = 0
			}
			afterMetrics[idx] = value
		}

		// Validate all assertions
		allPassed := true
		for idx, assertion := range testCase.Assertions {
			if assertion.Source != "metric" {
				continue
			}

			beforeVal := beforeMetrics[idx]
			afterVal := afterMetrics[idx]
			diff := afterVal - beforeVal

			if validateAssertion(diff, assertion.Expected.Operator, assertion.Expected.Value) {
				log.Printf("    • Assertion %d: ✓ (diff=%d, expected %s %d)", idx, diff, assertion.Expected.Operator, assertion.Expected.Value)
			} else {
				log.Printf("    • Assertion %d: ✗ (diff=%d, expected %s %d)", idx, diff, assertion.Expected.Operator, assertion.Expected.Value)
				allPassed = false
			}
		}

		if allPassed {
			validationPassed = true
			log.Printf("  ✓ Metrics validation passed on attempt %d\n", attempt)
			break
		}
	}

	if !validationPassed {
		log.Printf("  ✗ Metrics validation failed after %d attempts\n", *maxRetries)
	}

	// Count passed/failed assertions
	for idx, assertion := range testCase.Assertions {
		if assertion.Source != "metric" {
			result.PassedCount++
			continue
		}

		beforeVal := beforeMetrics[idx]
		afterVal := afterMetrics[idx]
		diff := afterVal - beforeVal

		if validateAssertion(diff, assertion.Expected.Operator, assertion.Expected.Value) {
			result.PassedCount++
		} else {
			result.FailedCount++
		}
	}

	if validationPassed && result.FailedCount == 0 {
		result.Status = "PASSED"
		result.Message = fmt.Sprintf("All %d assertions passed", result.AssertionCount)
	} else {
		result.Status = "FAILED"
		result.Message = fmt.Sprintf("%d passed, %d failed out of %d assertions", result.PassedCount, result.FailedCount, result.AssertionCount)
	}

	return result
}

// validateTestCase checks if the test case has all required fields
func validateTestCase(testCase TestCase) error {
	if testCase.PrometheusURL == "" {
		return fmt.Errorf("prometheus_url is required")
	}
	if testCase.ServiceEndpoint == "" {
		return fmt.Errorf("service_endpoint is required")
	}
	if testCase.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if testCase.VirtualService == "" && testCase.DestinationRule == "" {
		return fmt.Errorf("virtual_service or destination_rule is required")
	}
	if len(testCase.Assertions) == 0 {
		return fmt.Errorf("at least one assertion is required")
	}
	return nil
}

// queryMetric queries a metric from Prometheus
func queryMetric(prometheusURL, metricName string, labels map[string]string) (int, error) {
	switch metricName {
	case "istio_requests_total":
		return metrics.QueryIstioRequestsTotal(prometheusURL, labels)
	case "istio_tcp_connections_closed_total":
		return metrics.QueryIstioTcpConnectionsClosedTotal(prometheusURL, labels)
	default:
		return 0, fmt.Errorf("unsupported metric: %s", metricName)
	}
}

// validateAssertion checks if the actual value matches the expected condition
func validateAssertion(actualValue int, operator string, expectedValue int) bool {
	switch operator {
	case "eq":
		return actualValue == expectedValue
	case "gte":
		return actualValue >= expectedValue
	case "lte":
		return actualValue <= expectedValue
	case "gt":
		return actualValue > expectedValue
	case "lt":
		return actualValue < expectedValue
	default:
		log.Printf("Unknown operator: %s", operator)
		return false
	}
}

// printTestResult prints the result of a single test
func printTestResult(result TestResult) {
	statusIcon := colorGreen + "✓ PASSED" + colorReset
	borderChar := "┌"
	if result.Status == "FAILED" {
		statusIcon = colorRed + "✗ FAILED" + colorReset
		borderChar = "└"
	}

	fmt.Printf("\n%s─────────────────────────────────────────────────────────────────\n", borderChar)
	fmt.Printf("  %s | %s\n", statusIcon, result.TestName)
	fmt.Printf("  Assertions: %d passed, %d failed (total: %d)\n", result.PassedCount, result.FailedCount, result.AssertionCount)
	fmt.Printf("  %s\n", result.Message)
	fmt.Printf("─────────────────────────────────────────────────────────────────\n\n")
}

// printWelcomeBanner prints a welcome banner at the start
func printWelcomeBanner() {
	fmt.Printf("\n%s", colorGreen)
	fmt.Printf("╔═══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                                                               ║\n")
	fmt.Printf("║           OkResilience - Gateway Resilience Validator         ║\n")
	fmt.Printf("║                                                               ║\n")
	fmt.Printf("║        Automated Testing Suite for Istio Gateway Resilience   ║\n")
	fmt.Printf("║                                                               ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("%s\n", colorReset)
}

// printTestSummary prints overall test execution summary
func printTestSummary(results []TestResult) {
	totalTests := len(results)
	passedTests := 0
	failedTests := 0
	totalAssertions := 0
	passedAssertions := 0

	for _, result := range results {
		if result.Status == "PASSED" {
			passedTests++
		} else {
			failedTests++
		}
		totalAssertions += result.AssertionCount
		passedAssertions += result.PassedCount
	}

	// Summary banner
	fmt.Printf("\n%s", colorYellow)
	fmt.Printf("╔═══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    📊 EXECUTION SUMMARY                       ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("%s\n", colorReset)

	// Handle case where no tests were executed
	if totalTests == 0 {
		fmt.Printf("%s✗ No tests matched the specified filter%s\n", colorRed, colorReset)
		return
	}

	// Test results
	passIcon := colorGreen + "✓" + colorReset

	fmt.Printf("Tests:       %s %d/%d passed\n", passIcon, passedTests, totalTests)
	fmt.Printf("Assertions:  %s %d/%d passed\n", passIcon, passedAssertions, totalAssertions)

	// Progress bar
	barLength := 40
	var filled int
	var percentage int
	if totalTests > 0 {
		filled = (passedTests * barLength) / totalTests
		percentage = (passedTests * 100) / totalTests
	}
	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "="
		} else {
			bar += " "
		}
	}
	bar += "]"
	fmt.Printf("Progress:    %s %d%%\n", bar, percentage)

	fmt.Printf("\n")
	if passedTests == totalTests {
		fmt.Printf("%s╔═══════════════════════════════════════════════════════════════╗%s\n", colorGreen, colorReset)
		fmt.Printf("%s║  ✓ All tests PASSED! Gateway resilience verified successfully. ║%s\n", colorGreen, colorReset)
		fmt.Printf("%s╚═══════════════════════════════════════════════════════════════╝%s\n", colorGreen, colorReset)
	} else {
		fmt.Printf("%s╔═══════════════════════════════════════════════════════════════╗%s\n", colorRed, colorReset)
		fmt.Printf("%s║  ✗ %d test(s) FAILED - Review logs for details                  ║%s\n", colorRed, failedTests, colorReset)
		fmt.Printf("%s╚═══════════════════════════════════════════════════════════════╝%s\n", colorRed, colorReset)
	}
	fmt.Printf("\n")
}
