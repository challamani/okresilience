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

echo "Installing Istio with demo profile..."
  istioctl install --set profile=demo -y
  print_success "Istio installed successfully"

echo "Installing Kiali and Prometheus addons..."
  kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/kiali.yaml
  print_success "Kiali addon installed"

  kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/prometheus.yaml
  print_success "Prometheus addon installed"

echo "Configuring Gateways for Kiali and Prometheus..."
  kubectl apply -f resources/kiali-gateway.yaml
  print_success "Kiali gateway configured"
  kubectl apply -f resources/prometheus-gateway.yaml
  print_success "Prometheus gateway configured"

echo "Add hostname mapping in /etc/hosts, would require sudo access."
IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
print_success "Reach kiali.local prometheus.local on IP: $IP. \nPlease add the following entries to your /etc/hosts file if you access them via browser:"

echo "$IP kiali.local prometheus.local" | sudo tee -a /etc/hosts
print_success "Hostname mapping added to /etc/hosts"
echo "You can access Kiali at http://kiali.local and Prometheus at http://prometheus.local"