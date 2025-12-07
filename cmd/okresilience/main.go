package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/challamani/okresilience/pkg/metrics"
	"github.com/challamani/okresilience/pkg/traffic"
	"github.com/spf13/cobra"
)

const (
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

func main() {
	var prometheusURL, serviceEndpoint, namespace, virtualService, responseCode, app, sourceApp string
	var numRequests, delaySeconds, maxRetries int

	rootCmd := &cobra.Command{
		Use:   "okresilience",
		Short: "CLI tool to validate Kubernetes ingress gateway resilience",
		Long:  "OkResilience orchestrates traffic and Prometheus checks for upstream failure scenarios.",
	}

	// upstream5xxFailures subcommand encapsulates current functionality
	upstream5xxFailuresCmd := &cobra.Command{
		Use:   "upstream5xxFailures",
		Short: "Validate behavior for upstream 5xx responses",
		Run: func(cmd *cobra.Command, args []string) {
			// Fetch retries configuration
			retries, err := metrics.GetRetriesConfiguration(namespace, virtualService)
			if err != nil {
				log.Fatalf("Error fetching retries configuration: %v", err)
			}

			// Query metrics before traffic generation
			beforeRequests, err := metrics.QueryMetrics(prometheusURL, app, namespace, responseCode)
			if err != nil {
				log.Fatalf("Error querying metrics before traffic generation: %v", err)
			}
			log.Printf("Metrics before traffic generation: %d requests", beforeRequests)

			// Generate test traffic
			_, err = traffic.GenerateTraffic(serviceEndpoint, namespace, virtualService, numRequests)
			if err != nil {
				log.Fatalf("Error generating traffic: %v", err)
			}
			log.Printf("Traffic generation completed for service %s in namespace %s", virtualService, namespace)

			// Retry logic for fetching metrics after traffic generation
			var afterRequests int
			for attempt := 1; attempt <= maxRetries; attempt++ {
				log.Printf("Waiting %d seconds for metrics to sync (attempt %d/%d)...", delaySeconds, attempt, maxRetries)
				time.Sleep(time.Duration(delaySeconds) * time.Second)

				afterRequests, err = metrics.QueryMetrics(prometheusURL, app, namespace, responseCode)
				if err != nil {
					log.Printf("Error querying metrics after traffic generation (attempt %d): %v", attempt, err)
					continue
				}
				log.Printf("Metrics after traffic generation (attempt %d): %d requests", attempt, afterRequests)

				// Check if the difference in metrics matches the expected value
				actualRequests := afterRequests - beforeRequests
				expectedRequests := numRequests * (retries + 1)
				if actualRequests == expectedRequests {
					fmt.Printf("%sUpstream5xxFailures: Resilience validation completed successfully.%s\n", colorGreen, colorReset)
					return
				}

				log.Printf("Metrics validation failed (attempt %d): expected %d requests, got %d", attempt, expectedRequests, actualRequests)
			}

			// If we exhaust all retries, log the failure
			fmt.Printf("%sMetrics validation failed after %d attempts.%s\n", colorRed, maxRetries, colorReset)
		},
	}

	// (subcommands defined above)

	// upstreamTcpReset subcommand validates TCP upstream failures via destination tcp closed metrics
	upstreamTcpResetCmd := &cobra.Command{
		Use:   "upstreamTcpReset",
		Short: "Validate behavior for upstream TCP resets (UF/URX)",
		Run: func(cmd *cobra.Command, args []string) {
			// Fetch retries configuration
			retries, err := metrics.GetRetriesConfiguration(namespace, virtualService)
			if err != nil {
				log.Fatalf("Error fetching retries configuration: %v", err)
			}

			// Query destination TCP closed metrics before
			beforeClosed, err := metrics.QueryTcpClosedDest(prometheusURL, namespace, "UF,URX")
			if err != nil {
				log.Fatalf("Error querying TCP closed metrics before: %v", err)
			}
			log.Printf("TCP closed (dest) before: %d", beforeClosed)

			// Generate test traffic
			_, err = traffic.GenerateTraffic(serviceEndpoint, namespace, virtualService, numRequests)
			if err != nil {
				log.Fatalf("Error generating traffic: %v", err)
			}
			log.Printf("Traffic generation completed for service %s in namespace %s", virtualService, namespace)

			// Retry loop to wait for Prometheus to reflect metrics
			var afterClosed int
			for attempt := 1; attempt <= maxRetries; attempt++ {
				log.Printf("Waiting %d seconds for TCP metrics to sync (attempt %d/%d)...", delaySeconds, attempt, maxRetries)
				time.Sleep(time.Duration(delaySeconds) * time.Second)

				afterClosed, err = metrics.QueryTcpClosedDest(prometheusURL, namespace, "UF,URX")
				if err != nil {
					log.Printf("Error querying TCP closed metrics after (attempt %d): %v", attempt, err)
					continue
				}
				log.Printf("TCP closed (dest) after (attempt %d): %d", attempt, afterClosed)

				actualDest := afterClosed - beforeClosed
				expectedDest := numRequests * (retries + 1)

				if actualDest == expectedDest {
					fmt.Printf("%sUpstreamTcpReset: TCP upstream failure validation succeeded.%s\n", colorGreen, colorReset)
					return
				}
				log.Printf("Dest TCP validation failed (attempt %d): expected %d closed, got %d", attempt, expectedDest, actualDest)
			}

			fmt.Printf("%sTCP validation failed after %d attempts.%s\n", colorRed, maxRetries, colorReset)
		},
	}

	// Shared flags for all subcommands via root persistent flags
	rootCmd.PersistentFlags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus server URL")
	rootCmd.PersistentFlags().StringVar(&serviceEndpoint, "service-endpoint", "", "Service endpoint URL")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	rootCmd.PersistentFlags().StringVar(&virtualService, "virtual-service", "", "Name of the VirtualService resource")
	rootCmd.PersistentFlags().StringVar(&responseCode, "response-code", "200", "Expected HTTP response code for metrics validation")
	rootCmd.PersistentFlags().StringVar(&app, "app", "", "Destination application name for metrics query")
	rootCmd.PersistentFlags().IntVar(&numRequests, "num-requests", 10, "Number of requests to send to the service endpoint")
	rootCmd.PersistentFlags().IntVar(&delaySeconds, "delay-seconds", 5, "Delay in seconds to wait for metrics to sync")
	rootCmd.PersistentFlags().IntVar(&maxRetries, "max-retries", 5, "Maximum number of retries to fetch metrics after traffic generation")
	// Add source app flag for TCP validation
	rootCmd.PersistentFlags().StringVar(&sourceApp, "source-app", "", "Source application name for TCP metrics query (optional)")

	// gatewayTimeoutVerify subcommand
	gatewayTimeoutVerifyCmd := &cobra.Command{
		Use:   "gatewayTimeoutVerify",
		Short: "Verify Gateway timeout scenario (responseCode=0, downstream 504)",
		Run: func(cmd *cobra.Command, args []string) {
			// Fetch timeout and per-try timeout configuration
			timeout, perTryTimeout, err := metrics.GetTimeoutConfiguration(namespace, virtualService)
			if err != nil {
				log.Fatalf("Error fetching timeout configuration: %v", err)
			}

			// Query upstream metrics (responseCode=0) before traffic
			beforeTest, err := metrics.QueryMetrics(prometheusURL, app, namespace, "0")
			if err != nil {
				log.Fatalf("Error querying upstream metrics (responseCode=0) before: %v", err)
			}
			log.Printf("Upstream metrics (responseCode=0) before: %d", beforeTest)

			// Generate traffic and collect downstream response codes using traffic.GenerateTraffic
			downstreamCodes, err := traffic.GenerateTraffic(serviceEndpoint, namespace, virtualService, numRequests)
			if err != nil {
				log.Printf("Error generating traffic: %v", err)
			}
			num504 := 0
			for i, code := range downstreamCodes {
				log.Printf("Request %d: downstream status %d", i+1, code)
				if code == 504 {
					num504++
				}
			}
			log.Printf("Total downstream 504 responses: %d", num504)

			// Retry loop for upstream metrics
			var afterTest int
			for attempt := 1; attempt <= maxRetries; attempt++ {
				log.Printf("Waiting %d seconds for upstream metrics to sync (attempt %d/%d)...", delaySeconds, attempt, maxRetries)
				time.Sleep(time.Duration(delaySeconds) * time.Second)

				afterTest, err = metrics.QueryMetrics(prometheusURL, app, namespace, "0")
				if err != nil {
					log.Printf("Error querying upstream metrics (responseCode=0) after (attempt %d): %v", attempt, err)
					continue
				}
				log.Printf("Upstream metrics (responseCode=0) after (attempt %d): %d", attempt, afterTest)

				actual := afterTest - beforeTest
				ratio := timeout.Seconds() / perTryTimeout.Seconds()
				expected := int(math.Round(float64(numRequests) * ratio))
				if actual == expected && num504 == numRequests {
					fmt.Printf("%sGatewayTimeoutVerify: Validation succeeded. Upstream responseCode=0 diff: %d, downstream 504s: %d%s\n", colorGreen, actual, num504, colorReset)
					return
				}
				log.Printf("Validation failed (attempt %d): expected upstream diff %d, got %d; downstream 504s: %d/%d (timeout=%.2fs, perTryTimeout=%.2fs)", attempt, expected, actual, num504, numRequests, timeout.Seconds(), perTryTimeout.Seconds())
			}
			fmt.Printf("%sGatewayTimeoutVerify: Validation failed after %d attempts.%s\n", colorRed, maxRetries, colorReset)
		},
	}

	// Add subcommands at the end for readability
	rootCmd.AddCommand(upstream5xxFailuresCmd)
	rootCmd.AddCommand(upstreamTcpResetCmd)
	rootCmd.AddCommand(gatewayTimeoutVerifyCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
