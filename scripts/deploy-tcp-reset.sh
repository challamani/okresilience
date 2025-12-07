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

echo "Building Docker image for TCP Reset Service..."
docker build -t tcp-reset-service:local resources/tcp-reset
print_success "Docker image built successfully"

echo "Loading Docker image into Kind cluster..."
kind load docker-image tcp-reset-service:local --name ok-resilience
print_success "Docker image loaded into Kind cluster"

echo "Deploying TCP Reset Service to Kubernetes..."
kubectl apply -f resources/tcp-reset/deployment.yaml
print_success "TCP Reset Service deployed"

echo "Override existing virtual service to route traffic to TCP Reset Service..."
kubectl apply -f resources/tcp-reset/gateway.yaml
print_success "Virtual service configured for TCP Reset Service"

echo "Add hostname mapping in /etc/hosts, would require sudo access"
IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "$IP tcp-reset-service.local" | sudo tee -a /etc/hosts
print_success "Hostname mapping added to /etc/hosts"
echo "You can access tcp-reset-service at http://tcp-reset-service.local"