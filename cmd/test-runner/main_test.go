package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestLoadTestCases verifies that test cases can be loaded from JSON
func TestLoadTestCases(t *testing.T) {
	// Create a temporary test file
	testData := []TestCase{
		{
			Name:            "test1",
			Type:            "retry-5xx",
			PrometheusURL:   "http://prometheus:9090",
			ServiceEndpoint: "http://service:8080",
			Namespace:       "default",
			VirtualService:  "test-vs",
			NumRequests:     10,
			App:             "test-app",
			Assertions: []Assertion{
				{
					Source:     "metric",
					MetricName: "istio_requests_total",
					Labels:     map[string]string{"app": "test-app"},
					Expected: struct {
						Operator string `json:"operator"`
						Value    int    `json:"value"`
					}{Operator: "eq", Value: 10},
				},
			},
		},
	}

	data, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Error marshalling test data: %v", err)
	}

	tmpfile, err := os.CreateTemp("", "test-cases-*.json")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(data); err != nil {
		t.Fatalf("Error writing to temp file: %v", err)
	}
	tmpfile.Close()

	// Test loading
	testCases, err := loadTestCases(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error loading test cases: %v", err)
	}

	if len(testCases) != 1 {
		t.Errorf("Expected 1 test case, got %d", len(testCases))
	}

	if testCases[0].Name != "test1" {
		t.Errorf("Expected test name 'test1', got '%s'", testCases[0].Name)
	}

	if testCases[0].NumRequests != 10 {
		t.Errorf("Expected 10 requests, got %d", testCases[0].NumRequests)
	}
}

// TestValidateAssertion tests the assertion validation logic
func TestValidateAssertion(t *testing.T) {
	tests := []struct {
		name          string
		actualValue   int
		operator      string
		expectedValue int
		expected      bool
	}{
		{"eq_true", 10, "eq", 10, true},
		{"eq_false", 10, "eq", 5, false},
		{"gte_true", 10, "gte", 5, true},
		{"gte_equal", 10, "gte", 10, true},
		{"gte_false", 10, "gte", 15, false},
		{"lte_true", 10, "lte", 15, true},
		{"lte_equal", 10, "lte", 10, true},
		{"lte_false", 10, "lte", 5, false},
		{"gt_true", 10, "gt", 5, true},
		{"gt_false", 10, "gt", 10, false},
		{"lt_true", 10, "lt", 15, true},
		{"lt_false", 10, "lt", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAssertion(tt.actualValue, tt.operator, tt.expectedValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for operator '%s' with actual=%d, expected=%d",
					tt.expected, result, tt.operator, tt.actualValue, tt.expectedValue)
			}
		})
	}
}

// TestValidateTestCase validates test case validation logic
func TestValidateTestCase(t *testing.T) {
	tests := []struct {
		name       string
		testCase   TestCase
		shouldFail bool
	}{
		{
			name: "valid_test",
			testCase: TestCase{
				PrometheusURL:   "http://prometheus:9090",
				ServiceEndpoint: "http://service:8080",
				Namespace:       "default",
				VirtualService:  "vs",
				Assertions: []Assertion{
					{Source: "metric", MetricName: "istio_requests_total"},
				},
			},
			shouldFail: false,
		},
		{
			name: "missing_prometheus_url",
			testCase: TestCase{
				ServiceEndpoint: "http://service:8080",
				Namespace:       "default",
				VirtualService:  "vs",
			},
			shouldFail: true,
		},
		{
			name: "missing_service_endpoint",
			testCase: TestCase{
				PrometheusURL:  "http://prometheus:9090",
				Namespace:      "default",
				VirtualService: "vs",
			},
			shouldFail: true,
		},
		{
			name: "missing_namespace",
			testCase: TestCase{
				PrometheusURL:   "http://prometheus:9090",
				ServiceEndpoint: "http://service:8080",
				VirtualService:  "vs",
			},
			shouldFail: true,
		},
		{
			name: "missing_vs_and_dr",
			testCase: TestCase{
				PrometheusURL:   "http://prometheus:9090",
				ServiceEndpoint: "http://service:8080",
				Namespace:       "default",
			},
			shouldFail: true,
		},
		{
			name: "no_assertions",
			testCase: TestCase{
				PrometheusURL:   "http://prometheus:9090",
				ServiceEndpoint: "http://service:8080",
				Namespace:       "default",
				VirtualService:  "vs",
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTestCase(tt.testCase)
			if (err != nil) != tt.shouldFail {
				t.Errorf("Expected error=%v, got error=%v", tt.shouldFail, err != nil)
			}
		})
	}
}

// TestTestResultGeneration verifies test results are correctly generated
func TestTestResultGeneration(t *testing.T) {
	result := TestResult{
		TestName:       "test1",
		TestType:       "retry-5xx",
		Status:         "PASSED",
		Message:        "All assertions passed",
		AssertionCount: 2,
		PassedCount:    2,
		FailedCount:    0,
	}

	if result.Status != "PASSED" {
		t.Errorf("Expected status PASSED, got %s", result.Status)
	}

	if result.PassedCount != result.AssertionCount {
		t.Errorf("Expected all assertions to pass")
	}
}

// BenchmarkValidateAssertion benchmarks assertion validation
func BenchmarkValidateAssertion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		validateAssertion(100, "eq", 100)
	}
}

// TestAssertionStructure verifies assertion structure
func TestAssertionStructure(t *testing.T) {
	assertion := Assertion{
		Source:     "metric",
		MetricName: "istio_requests_total",
		Labels: map[string]string{
			"app":       "test-app",
			"namespace": "default",
		},
	}

	assertion.Expected.Operator = "eq"
	assertion.Expected.Value = 10

	if assertion.Source != "metric" {
		t.Errorf("Expected source 'metric', got %s", assertion.Source)
	}

	if assertion.Expected.Operator != "eq" {
		t.Errorf("Expected operator 'eq', got %s", assertion.Expected.Operator)
	}

	if assertion.Expected.Value != 10 {
		t.Errorf("Expected value 10, got %d", assertion.Expected.Value)
	}

	if assertion.Labels["app"] != "test-app" {
		t.Errorf("Expected app label 'test-app', got %s", assertion.Labels["app"])
	}
}
