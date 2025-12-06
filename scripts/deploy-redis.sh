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

echo "Installing redis deployment..."
kubectl apply -f resources/redis/deployment.yaml
print_success "redis deployment installed"

echo "Configuring Gateway, VirtualService for redis..."
kubectl apply -f resources/redis/gateway.yaml
print_success "redis virtual service configured"

echo "Add hostname mapping in /etc/hosts, would require sudo access"
IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Reach redis.local service on IP: $IP"
print_success "use: redis-cli -h $IP -p 31400 PING"