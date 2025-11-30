package traffic

import (
	"log"
	"net/http"
	"time"
)

// GenerateTraffic sends the specified number of test requests to the ingress gateway.
// It accepts the gateway URL, namespace, service name, and the number of requests to send.
func GenerateTraffic(gatewayURL, namespace, serviceName string, numRequests int) error {
	client := &http.Client{Timeout: 10 * time.Second}
	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(gatewayURL)
		if err != nil {
			log.Printf("Error sending request %d: %v", i+1, err)
			continue
		}
		log.Printf("Request %d: Received status code %d", i+1, resp.StatusCode)
		resp.Body.Close()
	}
	log.Printf("Traffic generation completed for service %s in namespace %s", serviceName, namespace)
	return nil
}
