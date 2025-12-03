#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Building Docker image for TCP Reset Service..."
docker build -t tcp-reset-service:local resources/tcp-reset-service

echo "Loading Docker image into Kind cluster..."
kind load docker-image tcp-reset-service:local --name ok-resilience

echo "Deploying TCP Reset Service to Kubernetes..."
kubectl apply -f resources/tcp-reset-service/deployment.yaml

echo "Override existing virtual service to route traffic to TCP Reset Service..."
kubectl apply -f resources/tcp-reset-service/tcp-reset-virtualservice.yaml
echo "TCP Reset Service deployed successfully."