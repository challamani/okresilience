package traffic

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateTraffic(t *testing.T) {
	// Mock HTTP server to simulate the ingress gateway
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Simulate a successful response
	}))
	defer mockServer.Close()

	tests := []struct {
		name        string
		gatewayURL  string
		namespace   string
		serviceName string
		numRequests int
		expectError bool
	}{
		{
			name:        "Valid traffic generation",
			gatewayURL:  mockServer.URL,
			namespace:   "demo",
			serviceName: "test-service",
			numRequests: 5,
			expectError: false,
		},
		{
			name:        "Invalid gateway URL",
			gatewayURL:  "http://invalid-url",
			namespace:   "demo",
			serviceName: "test-service",
			numRequests: 3,
			expectError: true,
		},
		{
			name:        "Zero requests",
			gatewayURL:  mockServer.URL,
			namespace:   "demo",
			serviceName: "test-service",
			numRequests: 0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call GenerateTraffic
			_, err := GenerateTraffic(tt.gatewayURL, tt.namespace, tt.serviceName, tt.numRequests)

			// Check if an error was expected
			if (err != nil) != tt.expectError {
				t.Errorf("GenerateTraffic() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
