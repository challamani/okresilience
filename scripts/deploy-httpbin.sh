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

echo "Installing httpbin deployment..."
kubectl apply -f resources/httpbin/deployment.yaml
print_success "httpbin deployment installed"

echo "Configuring Gateway, VirtualService and DestinationRule for httpbin..."
kubectl apply -f resources/httpbin/gateway.yaml
print_success "httpbin virtual service configured"

echo "Add hostname mapping in /etc/hosts, would require sudo access"
IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "$IP httpbin.local" | sudo tee -a /etc/hosts
print_success "Hostname mapping added to /etc/hosts"
echo "You can access httpbin at http://httpbin.local"