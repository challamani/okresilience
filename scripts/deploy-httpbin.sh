#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Installing httpbin deployment..."
kubectl apply -f resources/httpbin/httpbin.yaml

echo "Configuring Gateway and VirtualService for httpbin..."
kubectl apply -f resources/httpbin/httpbin-gateway.yaml
kubectl apply -f resources/httpbin/httpbin-virtualservice.yaml

echo "Configuring DestinationRule for httpbin..."
kubectl apply -f resources/httpbin/httpbin-destinationrule.yaml

echo "Add hostname mapping in /etc/hosts, would require sudo access"
IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "$IP httpbin.local" | sudo tee -a /etc/hosts
echo "httpbin installed successfully."
echo "You can access httpbin at http://httpbin.local"