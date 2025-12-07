# Ok Resilience

CLI tool for testing Gateway resilience

## Overview

This tool is designed to help validate the resilience of API Gateways (such as Istio Ingress Gateway) by simulating various failure scenarios and observing how the gateway handles them. It focuses on two main types of failures.

- Simulating TCP Failures (e.g., connection failures, resets)
- Simulating HTTP Failures (e.g., gateway failures, upstream timeouts, retryable 5XX errors for GET requests)

Assert the gateway's behavior under these failure conditions by checking metrics from Prometheus.

Tool supports the following resilience tests:

- Gateway Retries on Upstream 5xx Failures
- Gateway Retries on Upstream TCP Resets
- Outlier Detection and Ejection (Circuit Breaking)  
- Validate Gateway Per Request Timeouts
- Failover Testing for Multi-Cluster Gateways

## Prerequisites

Ensure the following tools are installed on your system:

- [Homebrew](https://brew.sh/) (for macOS/Linux users)
- [Go](https://golang.org/) (version 1.18 or later)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [istioctl](https://istio.io/latest/docs/setup/getting-started/)
- [kind](https://kind.sigs.k8s.io/)
- [cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind)

### Install Prerequisites

For macOS/Linux users, you can install the tools using Homebrew:

```bash
brew install go
brew install kubectl
brew install istioctl
brew install kind
brew install cloud-provider-kind
```

For other platforms, refer to the official installation guides for each tool.

### Setting Up the Environment

### Httpbin in Kind Cluster

To set up the `httpbin` service in a local Kubernetes cluster using Kind, follow these steps:

- Create a Kind cluster if you don't have one already:

```bash
./scripts/setup-kind.sh
```

- Setup Istio, Kiali, and Prometheus:

```bash
# Install Istio using istioctl, then Kiali and Prometheus.
./scripts/install-istio.sh
```

- Deploy `httpbin` application:

```bash
./scripts/deploy-httpbin.sh
```

- Generate some traffic to httpbin:

```bash
export INGRESS_IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
for i in {1..50}; do echo -e "\nRequest ==> [$i]"; curl -s -D - -o /dev/null -H "Host: httpbin.local" http://$INGRESS_IP/get; sleep 1; done

# use generate traffic script
./scripts/generate-traffic.sh
```

## Setup

- Install dependencies:

```bash
go mod tidy
```

- Build the CLI tool:

```bash
go build -o okresilience ./cmd/okresilience
```

## Run

### Validate upstream 5xx failures with retries

With this test you can validate that the gateway retries requests when upstream services return 5xx errors.

```bash
./okresilience upstream5xxFailures \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://httpbin.local/status/500 \
    --namespace=demo \
    --virtual-service=httpbin-vs \
    --num-requests=1 \
    --response-code=500 \
    --app=httpbin
```

### Validate upstream TCP resets

With this test you can validate that the gateway retries requests when TCP resets occur in upstream services.

```bash
./okresilience upstreamTcpReset \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://tcp-reset-service.local/ \
    --namespace=tcp-ns \
    --virtual-service=tcp-reset-vs \
    --num-requests=1 \
    --response-code=503 \
    --source-app=istio-ingressgateway
```

## Running Unit Tests

```bash
go test ./... -v
```

## TCP Failure Simulation

Deploy the `tcp-reset-service` for testing TCP failures:

```bash
./scripts/deploy-tcp-reset.sh
```
