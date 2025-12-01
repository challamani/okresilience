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
# Install Istio using istioctl
istioctl install --set profile=demo -y
./scripts/install-kiali.sh
```

- Deploy httpbin application:

```bash
./scripts/deploy-httpbin.sh
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

./okresilience --prometheus-url=$PROMETHEUS_URL \
    --service-endpoint=$SERVICE_ENDPOINT \
    --namespace=$NAMESPACE \
    --virtual-service=$VIRTUAL_SERVICE \
    --num-requests=1 \
    --response-code=$RESPONSE_CODE \
    --app=$APP
```

### Notes

- The `--response-code` flag specifies the expected HTTP response code for metrics validation (default: `200`).
- The CLI queries Prometheus metrics for the last 5 minutes to validate resilience settings.

## Running Unit Tests

To run the unit tests for the project, use the following command:

```bash
#This will execute all tests in the project and display detailed output.
go test ./... -v
```