package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	istioClient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// PrometheusResponse represents the structure of Prometheus query results.
type PrometheusResponse struct {
	Data struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// ValidateMetrics queries Prometheus and validates the metrics.
func ValidateMetrics(prometheusURL, namespace string, expectedRequests int) error {
	query := fmt.Sprintf(`istio_requests_total{namespace="%s"}`, namespace)
	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, query)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error querying Prometheus: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading Prometheus response: %v", err)
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return fmt.Errorf("error unmarshalling Prometheus response: %v", err)
	}

	// Extract metrics and validate
	totalRequests := 0
	for _, result := range promResp.Data.Result {
		if value, ok := result.Value[1].(string); ok {
			metricValue, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("error converting metric value: %v", err)
			}
			totalRequests += metricValue
		}
	}

	if totalRequests != expectedRequests {
		return fmt.Errorf("metrics validation failed: expected %d requests, got %d", expectedRequests, totalRequests)
	}

	log.Printf("Metrics validation successful: %d requests matched", totalRequests)
	return nil
}

// GetRetriesConfiguration fetches the retries configuration from the VirtualService.
func GetRetriesConfiguration(namespace, serviceName string) (int, error) {
	// Create Kubernetes configuration
	config, err := getKubernetesConfig()
	if err != nil {
		return 0, fmt.Errorf("error creating Kubernetes config: %v", err)
	}

	// Create an Istio clientset
	istioClientset, err := istioClient.NewForConfig(config)
	if err != nil {
		return 0, fmt.Errorf("error creating Istio client: %v", err)
	}

	// Fetch the VirtualService resource
	vs, err := istioClientset.NetworkingV1alpha3().VirtualServices(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("error fetching VirtualService: %v", err)
	}

	// Extract the retries configuration
	if vs.Spec.Http == nil || len(vs.Spec.Http) == 0 {
		return 0, fmt.Errorf("no HTTP routes found in VirtualService")
	}

	if vs.Spec.Http[0].Retries == nil {
		return 0, fmt.Errorf("no retries configuration found in VirtualService")
	}

	return int(vs.Spec.Http[0].Retries.Attempts), nil
}

// getKubernetesConfig attempts to load in-cluster configuration, falling back to out-of-cluster configuration.
func getKubernetesConfig() (*rest.Config, error) {
	// Try in-cluster configuration
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Println("Using in-cluster Kubernetes configuration")
		return config, nil
	}

	// Fallback to out-of-cluster configuration
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to load Kubernetes configuration: %v", err)
	}

	log.Println("Using out-of-cluster Kubernetes configuration")
	return config, nil
}
