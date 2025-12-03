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

## Build

```bash
go build -o okresilience ./cmd/okresilience
```

## Run

```bash
PROMETHEUS_URL=http://prometheus.local
SERVICE_ENDPOINT=http://httpbin.local/status/500
NAMESPACE=demo
VIRTUAL_SERVICE=httpbin-vs
RESPONSE_CODE=500
APP=httpbin

./okresilience upstream5xxFailures --prometheus-url=$PROMETHEUS_URL \
    --service-endpoint=$SERVICE_ENDPOINT \
    --namespace=$NAMESPACE \
    --virtual-service=$VIRTUAL_SERVICE \
    --num-requests=1 \
    --response-code=$RESPONSE_CODE \
    --app=$APP
```

### Notes

- The `--response-code` flag specifies the expected HTTP response code for metrics validation (default: `200`).
- The CLI queries Prometheus metrics before and after the test traffic execution and calculates the difference.
- The difference is used to validate if the gateway retries are functioning as expected.

## Running Unit Tests

## TCP Failure Simulation

See `resources/tcp-reset-service/` for a simple service that emits TCP RST packets to simulate connection failures.

Quick start:

```zsh
# This script builds and deploys the tcp-reset-service
./scripts/deploy-tcp-reset-service.sh
```

**Note**: TCP-level failures won't appear in `istio_requests_total` metrics (no HTTP request is formed). Look for client-side connection errors and `istio_tcp_*` metrics instead. See `resources/tcp-reset-service/README.md` for details.
To run the unit tests for the project, use the following command:

```bash
#This will execute all tests in the project and display detailed output.
go test ./... -v
```

```shell
./okresilience upstreamTcpReset --prometheus-url=$PROMETHEUS_URL \
    --service-endpoint=$SERVICE_ENDPOINT \
    --namespace=$NAMESPACE \
    --virtual-service=$VIRTUAL_SERVICE \
    --num-requests=1 \
    --response-code=$RESPONSE_CODE \
    --app=$APP

```