package main

import (
	"fmt"
	"log"
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
	var prometheusURL, serviceEndpoint, namespace, virtualService, responseCode, app string
	var numRequests, delaySeconds, maxRetries int

	rootCmd := &cobra.Command{
		Use:   "okresilience",
		Short: "CLI tool to validate Kubernetes ingress gateway resilience",
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
			err = traffic.GenerateTraffic(serviceEndpoint, namespace, virtualService, numRequests)
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
					fmt.Printf("%sResilience validation completed successfully.%s\n", colorGreen, colorReset)
					return
				}

				log.Printf("Metrics validation failed (attempt %d): expected %d requests, got %d", attempt, expectedRequests, actualRequests)
			}

			// If we exhaust all retries, log the failure
			fmt.Printf("%sMetrics validation failed after %d attempts.%s\n", colorRed, maxRetries, colorReset)
		},
	}

	rootCmd.Flags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus server URL")
	rootCmd.Flags().StringVar(&serviceEndpoint, "service-endpoint", "", "Service endpoint URL")
	rootCmd.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	rootCmd.Flags().StringVar(&virtualService, "virtual-service", "", "Name of the VirtualService resource")
	rootCmd.Flags().StringVar(&responseCode, "response-code", "200", "Expected HTTP response code for metrics validation")
	rootCmd.Flags().StringVar(&app, "app", "", "Destination application name for metrics query")
	rootCmd.Flags().IntVar(&numRequests, "num-requests", 10, "Number of requests to send to the service endpoint")
	rootCmd.Flags().IntVar(&delaySeconds, "delay-seconds", 5, "Delay in seconds to wait for metrics to sync")
	rootCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "Maximum number of retries to fetch metrics after traffic generation")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
