package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"k8s.io/client-go/rest"
)

func TestQueryMetrics(t *testing.T) {
	// Mock Prometheus server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query == `istio_requests_total{destination_app="test-app",namespace="demo",response_code="200"}` {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"result": [
						{
							"metric": {"response_code": "200"},
							"value": [1680000000, "15"]
						}
					]
				}
			}`))
		} else {
			// Return valid JSON for invalid queries to avoid unmarshalling errors
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"result": []
				}
			}`))
		}
	}))
	defer mockServer.Close()

	tests := []struct {
		name          string
		prometheusURL string
		app           string
		namespace     string
		responseCode  string
		expected      int
		expectError   bool
	}{
		{
			name:          "Valid query",
			prometheusURL: mockServer.URL,
			app:           "test-app",
			namespace:     "demo",
			responseCode:  "200",
			expected:      15,
			expectError:   false,
		},
		{
			name:          "Invalid query",
			prometheusURL: mockServer.URL,
			app:           "invalid-app",
			namespace:     "demo",
			responseCode:  "200",
			expected:      0,
			expectError:   false,
		},
		{
			name:          "Invalid Prometheus URL",
			prometheusURL: "http://invalid-url",
			app:           "test-app",
			namespace:     "demo",
			responseCode:  "200",
			expected:      0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := QueryMetrics(tt.prometheusURL, tt.app, tt.namespace, tt.responseCode)
			if (err != nil) != tt.expectError {
				t.Errorf("QueryMetrics() error = %v, expectError %v", err, tt.expectError)
			}
			if result != tt.expected {
				t.Errorf("QueryMetrics() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetRetriesConfiguration(t *testing.T) {
	// Mock Kubernetes API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // Set correct content type
		if r.URL.Path == "/apis/networking.istio.io/v1alpha3/namespaces/demo/virtualservices/test-service" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"spec": {
					"http": [
						{
							"retries": {
								"attempts": 3
							}
						}
					]
				}
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Override getKubernetesConfig to use the mock server
	originalGetKubernetesConfig := getKubernetesConfig
	getKubernetesConfig = func() (*rest.Config, error) {
		return &rest.Config{Host: mockServer.URL}, nil
	}
	defer func() { getKubernetesConfig = originalGetKubernetesConfig }() // Restore original function after test

	tests := []struct {
		name        string
		namespace   string
		serviceName string
		expected    int
		expectError bool
	}{
		{
			name:        "Valid VirtualService",
			namespace:   "demo",
			serviceName: "test-service",
			expected:    3,
			expectError: false,
		},
		{
			name:        "VirtualService not found",
			namespace:   "demo",
			serviceName: "nonexistent-service",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetRetriesConfiguration(tt.namespace, tt.serviceName)
			if (err != nil) != tt.expectError {
				t.Errorf("GetRetriesConfiguration() error = %v, expectError %v", err, tt.expectError)
			}
			if result != tt.expected {
				t.Errorf("GetRetriesConfiguration() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
