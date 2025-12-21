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
	"sort"
	"strconv"
	"strings"
	"time"

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

// formatLabelSet turns a label map into a deterministic PromQL label selector string.
// Keys are sorted to keep query strings stable for testing and logging.
func formatLabelSet(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(labels))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, labels[k]))
	}

	return strings.Join(parts, ",")
}

// QueryIstioRequestsTotal queries the istio_requests_total metric using the provided label set.
// Labels are passed as key/value pairs to support dynamic label selection at the call site.
func QueryIstioRequestsTotal(prometheusURL string, labels map[string]string) (int, error) {
	return queryPrometheusTotal(prometheusURL, "istio_requests_total", labels)
}

// QueryIstioTcpConnectionsClosedTotal queries the istio_tcp_connections_closed_total metric using the provided label set.
// Labels are passed as key/value pairs to support dynamic label selection at the call site.
func QueryIstioTcpConnectionsClosedTotal(prometheusURL string, labels map[string]string) (int, error) {
	return queryPrometheusTotal(prometheusURL, "istio_tcp_connections_closed_total", labels)
}

// queryPrometheusTotal executes a Prometheus instant query for the given metric and labels and
// returns the summed integer value across all returned series.
func queryPrometheusTotal(prometheusURL, metric string, labels map[string]string) (int, error) {
	labelExpr := formatLabelSet(labels)
	query := metric
	if labelExpr != "" {
		query = fmt.Sprintf("%s{%s}", metric, labelExpr)
	}

	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, query)
	log.Printf("Executing Prometheus query: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("error querying Prometheus: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading Prometheus response: %v", err)
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return 0, fmt.Errorf("error unmarshalling Prometheus response: %v", err)
	}

	if len(promResp.Data.Result) == 0 {
		log.Printf("No metrics found for query: %s", query)
		return 0, nil
	}

	total := 0
	for _, result := range promResp.Data.Result {
		if len(result.Value) < 2 {
			log.Printf("Skipping result with insufficient Value data: %v", result.Value)
			continue
		}

		valueStr, ok := result.Value[1].(string)
		if !ok {
			log.Printf("Skipping result with invalid Value data: %v", result.Value)
			continue
		}

		valFloat, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			log.Printf("Skipping result with non-float Value: %v", result.Value)
			continue
		}

		total += int(valFloat)
	}

	return total, nil
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
	if len(vs.Spec.Http) == 0 {
		return 0, fmt.Errorf("no HTTP routes found in VirtualService")
	}

	if vs.Spec.Http[0].Retries == nil {
		return 0, fmt.Errorf("no retries configuration found in VirtualService")
	}

	return int(vs.Spec.Http[0].Retries.Attempts), nil
}

// GetTimeoutConfiguration returns the HTTP route timeout and per-try timeout durations from a VirtualService.
func GetTimeoutConfiguration(namespace, serviceName string) (time.Duration, time.Duration, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return 0, 0, fmt.Errorf("error creating Kubernetes config: %v", err)
	}

	istioClientset, err := istioClient.NewForConfig(config)
	if err != nil {
		return 0, 0, fmt.Errorf("error creating Istio client: %v", err)
	}

	vs, err := istioClientset.NetworkingV1alpha3().VirtualServices(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("error fetching VirtualService: %v", err)
	}

	if len(vs.Spec.Http) == 0 {
		return 0, 0, fmt.Errorf("no HTTP routes found in VirtualService")
	}

	route := vs.Spec.Http[0]
	if route.Timeout == nil {
		return 0, 0, fmt.Errorf("timeout not configured in VirtualService")
	}

	timeout := route.Timeout.AsDuration()
	if route.Retries == nil || route.Retries.PerTryTimeout == nil {
		return timeout, 0, fmt.Errorf("perTryTimeout not configured in VirtualService")
	}

	perTryTimeout := route.Retries.PerTryTimeout.AsDuration()
	if perTryTimeout == 0 {
		return timeout, 0, fmt.Errorf("perTryTimeout is zero in VirtualService")
	}

	return timeout, perTryTimeout, nil
}

func GetOutlierDetectionConfiguration(namespace, serviceName string) (int32, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return 0, fmt.Errorf("error creating Kubernetes config: %v", err)
	}
	istioClientset, err := istioClient.NewForConfig(config)
	if err != nil {
		return 0, fmt.Errorf("error creating Istio client: %v", err)
	}
	destinationRule, err := istioClientset.NetworkingV1beta1().DestinationRules(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil { 
		return 0, fmt.Errorf("error fetching DestinationRule: %v", err)
	}

	consecutive5xxErrors := destinationRule.Spec.TrafficPolicy.OutlierDetection.Consecutive_5XxErrors.Value
	if consecutive5xxErrors == 0 {
		return 0, fmt.Errorf("consecutive5xxErrors not configured in DestinationRule")
	}
	return int32(consecutive5xxErrors), nil
}
// getKubernetesConfig is a variable that holds a function to fetch Kubernetes configuration.
// It can be overridden in tests to mock the behavior.
var getKubernetesConfig = func() (*rest.Config, error) {
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
