#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Installing Istio with demo profile..."
istioctl install --set profile=demo -y

echo "Installing Kiali and Prometheus addons..."
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/kiali.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/prometheus.yaml

echo "Kiali and Prometheus addons installed successfully."

echo "Configuring Gateways for Kiali and Prometheus..."
kubectl apply -f resources/kiali-gateway.yaml
kubectl apply -f resources/prometheus-gateway.yaml

echo "Add hostname mapping in /etc/hosts, would require sudo access."
IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "$IP kiali.local prometheus.local" | sudo tee -a /etc/hosts
echo "You can access Kiali at http://kiali.local and Prometheus at http://prometheus.local"