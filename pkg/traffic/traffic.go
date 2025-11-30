package traffic

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// GenerateTraffic sends the specified number of test requests to the ingress gateway.
// It accepts the gateway URL, namespace, service name, and the number of requests to send.
func GenerateTraffic(gatewayURL, namespace, serviceName string, numRequests int) error {
	// Treat zero (or negative) requests as a no-op (idempotent behavior).
	if numRequests <= 0 {
		log.Printf("No traffic generated (numRequests=%d) for service %s in namespace %s", numRequests, serviceName, namespace)
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var failedRequests int

	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(gatewayURL)
		if err != nil {
			log.Printf("Error sending request %d: %v", i+1, err)
			failedRequests++
			continue
		}
		log.Printf("Request %d: Received status code %d", i+1, resp.StatusCode)
		resp.Body.Close()
	}

	log.Printf("Traffic generation completed for service %s in namespace %s", serviceName, namespace)

	// Only error if all attempts failed AND at least one was attempted.
	if failedRequests == numRequests && numRequests > 0 {
		return fmt.Errorf("all requests failed during traffic generation")
	}

	return nil
}
