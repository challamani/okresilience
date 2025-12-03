# Ok Resilience

CLI tool for testing Gateway resilience

Planned Features (Coming Soon):

- Simulating TCP Failures (e.g., connection failures, resets)
- Simulating HTTP Failures (e.g., gateway failures, upstream timeouts, retryable 5XX errors for GET requests)
- Gateway Retries
- Outlier Detection
- Circuit Breaking

## Prerequisites

Ensure the following tools are installed on your system:

- [Homebrew](https://brew.sh/) (for macOS/Linux users)
- [Go](https://golang.org/) (version 1.18 or later)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [istioctl](https://istio.io/latest/docs/setup/getting-started/)
- [kind](https://kind.sigs.k8s.io/)

### Install Prerequisites

For macOS/Linux users, you can install the tools using Homebrew:

```bash
brew install go
brew install kubectl
brew install istioctl
brew install kind
```

For other platforms, refer to the official installation guides for each tool.

### Setting Up the Environment

### Httpbin in Kind Cluster

To set up the `httpbin` service in a local Kubernetes cluster using Kind, follow these steps:

- Create a Kind cluster if you don't have one already:

```bash
./scripts/kind-setup.sh
```

- Setup Istio, Kiali, and Prometheus:

```bash
# Install Istio using istioctl, then Kiali and Prometheus.
./scripts/install-istio.sh
```

- Deploy httpbin application:

```bash
./scripts/deploy-httpbin.sh
```

- Generate some traffic to httpbin:

```bash
for i in {1..10}; do curl -s -D - -o /dev/null http://httpbin.local/status/200; done
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

```bash
./okresilience upstreamTcpReset \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://httpbin.local/status/200 \
    --namespace=demo \
    --virtual-service=httpbin-vs \
    --num-requests=1 \
    --response-code=503 \
    --source-app=istio-ingressgateway
```

## Running Unit Tests

```bash
go test ./... -v
```

## TCP Failure Simulation

Deploy the tcp-reset-service for testing TCP failures:

```bash
./scripts/deploy-tcp-reset-service.sh
```
