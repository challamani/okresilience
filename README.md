# Ok Resilience

CLI tool for testing Gateway resilience

## Features

- **Traffic Generation**: Sends a configurable number of test requests to the service endpoint.
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
PROMETHEUS_URL=http://prometheus.local
SERVICE_ENDPOINT=http://httpbin.local/status/200
NAMESPACE=demo
VIRTUAL_SERVICE=httpbin-vs
RESPONSE_CODE=500

./okresilience --prometheus-url=$PROMETHEUS_URL \
    --service-endpoint=$SERVICE_ENDPOINT \
    --namespace=$NAMESPACE \
    --virtual-service=$VIRTUAL_SERVICE \
    --response-code=$RESPONSE_CODE \
    --num-requests=2
```

### Notes

- The `--response-code` flag specifies the expected HTTP response code for metrics validation (default: `200`).
- The CLI queries Prometheus metrics for the last 5 minutes to validate resilience settings.
