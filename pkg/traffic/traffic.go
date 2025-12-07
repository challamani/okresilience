package traffic

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// GenerateTraffic sends the specified number of test requests to the ingress gateway.
// It accepts the gateway URL, namespace, service name, and the number of requests to send.
// GenerateTraffic sends the specified number of test requests to the ingress gateway.
// Returns a slice of response codes for each request, and error if all failed.
func GenerateTraffic(gatewayURL, namespace, serviceName string, numRequests int) ([]int, error) {
	if numRequests <= 0 {
		log.Printf("No traffic generated (numRequests=%d) for service %s in namespace %s", numRequests, serviceName, namespace)
		return []int{}, nil
	}

	client := &http.Client{Timeout: 60 * time.Second}
	var failedRequests int
	var codes []int

	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(gatewayURL)
		if err != nil {
			log.Printf("Error sending request %d: %v", i+1, err)
			failedRequests++
			codes = append(codes, 0)
			continue
		}
		log.Printf("Request %d: Received status code %d", i+1, resp.StatusCode)
		codes = append(codes, resp.StatusCode)
		resp.Body.Close()
	}

	log.Printf("Traffic generation completed for service %s in namespace %s", serviceName, namespace)

	if failedRequests == numRequests && numRequests > 0 {
		return codes, fmt.Errorf("all requests failed during traffic generation")
	}

	return codes, nil
}
