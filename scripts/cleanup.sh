#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Color codes
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Function to print success message with green tick
print_success() {
  echo -e "${GREEN}✓${NC} $1"
}

# Function to clean up Kubernetes resources
cleanup_kubernetes_resources() {
  echo "Cleaning up Kubernetes resources..."
  kubectl delete -f resources/httpbin/deployment.yaml || echo "Kubernetes resources not found or already deleted."
  print_success "httpbin resources cleaned up"
  kubectl delete -f resources/tcp-reset-service/deployment.yaml || echo "Kubernetes resources not found or already deleted."
  print_success "TCP reset service resources cleaned up"
}

# Function to delete the Kind cluster
delete_kind_cluster() {
  echo "Deleting Kind cluster..."
  kind delete cluster --name ok-resilience || echo "Kind cluster not found or already deleted."
  print_success "Kind cluster deleted successfully"
}

# Main script execution
echo "Starting cleanup process..."
cleanup_kubernetes_resources
delete_kind_cluster
print_success "Cleanup process completed"
