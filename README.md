# Ok Resilience

CLI tool for testing Gateway resilience

Planned Features (Coming Soon):

- Simulating TCP Failures (e.g., connection failures, resets)
- Simulating HTTP Failures (e.g., gateway failures, upstream timeouts, retryable 5XX errors for GET requests)
- Outlier Detection
- Circuit Breaking

## Prerequisites

- Docker
- Kind/Minikube
- istioctl

## Create a cluster using kind

```shell
#create a kind cluster with name 'okresilience'
./scripts/kind-setup.sh
```

## Install istio

```shell
istioctl install --set profile=demo -y
```

## Install Kiali Dashboard

- Install Kiali with Prometheus in `istio-system` namespace

```shell
./scripts/install-kiali.sh
```

- [Access Kiali](http://kiali.local/)

## Deploy a sample httpbin application

```shell
./scripts/deploy-httpbin.sh
```

### generate traffic to httpbin

[Launch httpbin in browser](http://httpbin.local)

```shell
for i in {1..10}; do curl -s -D - -o /dev/null  "http://httpbin.local/status/200"; done
```

# OkResilience

CLI tool to validate Kubernetes ingress gateway resilience.

## Features

- **Traffic Generation**: Sends a configurable number of test requests to the ingress gateway.
- **Metrics Validation**: Queries Prometheus metrics to validate resilience settings.
- **Retries Configuration**: Reads retries configuration from the VirtualService resource.

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
./okresilience --prometheus-url=http://<PROMETHEUS_URL> --gateway-url=http://<GATEWAY_URL> --namespace=demo --service-name=httpbin --num-requests=10
```
