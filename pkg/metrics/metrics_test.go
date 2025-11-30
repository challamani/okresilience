package metrics
package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateMetrics(t *testing.T) {
	// Mock Prometheus server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") == `istio_requests_total{namespace="demo",response_code="500"}` {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"result": [
						{
							"metric": {"response_code": "500"},
							"value": [1680000000, "5"]
						}
					]
				}
			}`))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer mockServer.Close()

	tests := []struct {
		name            string
		prometheusURL   string
		namespace       string
		expectedRequests int
		responseCode    string
		timeRange       string
		expectError     bool
	}{
		{
			name:            "Valid metrics query",
			prometheusURL:   mockServer.URL,
			namespace:       "demo",
			expectedRequests: 5,
			responseCode:    "500",
			timeRange:       "1680000000",
			expectError:     false,
		},
		{
			name:            "Invalid metrics query",
			prometheusURL:   mockServer.URL,
			namespace:       "demo",
			expectedRequests: 10,
			responseCode:    "500",
			timeRange:       "1680000000",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetrics(tt.prometheusURL, tt.namespace, tt.expectedRequests, tt.responseCode, tt.timeRange)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateMetrics() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
