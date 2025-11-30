#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to install prerequisites
install_prerequisites() {
  echo "Installing prerequisites..."
  brew install kind || echo "Kind is already installed."
  brew install cloud-provider-kind || echo "cloud-provider-kind is already installed."
  echo "Prerequisites installed successfully."
}

# Function to create a Kind cluster
create_kind_cluster() {
  echo "Creating Kind cluster..."
  kind create cluster --name ok-resilience
  echo "Kind cluster created successfully."
}

# Function to activate cloud-provider-kind
activate_cloud_provider_kind() {
  #[Load Balancer](https://kind.sigs.k8s.io/docs/user/loadbalancer/) service
  echo "Activating cloud-provider-kind..."
  sudo cloud-provider-kind --gateway-channel standard
}

# Main script execution
echo "Starting Kind setup..."
install_prerequisites
create_kind_cluster
activate_cloud_provider_kind
