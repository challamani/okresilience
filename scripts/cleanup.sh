#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to clean up Kubernetes resources
cleanup_kubernetes_resources() {
  echo "Cleaning up Kubernetes resources..."
  kubectl delete -f resources/httpbin/httpbin.yaml || echo "Kubernetes resources not found or already deleted."
  kubectl delete -f resources/tcp-reset-service/deployment.yaml || echo "Kubernetes resources not found or already deleted."
  echo "Kubernetes resources cleaned up successfully."
}

# Function to delete the Kind cluster
delete_kind_cluster() {
  echo "Deleting Kind cluster..."
  kind delete cluster --name ok-resilience || echo "Kind cluster not found or already deleted."
  echo "Kind cluster deleted successfully."
}

# Main script execution
echo "Starting cleanup process..."
cleanup_kubernetes_resources
delete_kind_cluster
echo "Cleanup process completed."
