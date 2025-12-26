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
./scripts/generate-traffic.sh            # defaults to status endpoint
./scripts/generate-traffic.sh header     # hit /get to echo headers
./scripts/generate-traffic.sh delay      # hit /delay/5 to simulate slow upstreams

REQUEST_COUNT=10
./scripts/generate-traffic.sh status  # override request volume
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

- Build the resilience executor:

```bash
go build -o resilience-executor ./cmd/test-runner
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
    --app=tcp-reset-service
```

### Validate Gateway timeout behavior

Use this to verify that ingress timeouts surface as upstream `responseCode=0` while clients observe HTTP 504. The upstream diff should equal `num-requests * (timeout / perTryTimeout)` as configured in the VirtualService.

```bash
./okresilience gatewayTimeoutVerify \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://httpbin.local/delay/3 \
    --namespace=demo \
    --virtual-service=httpbin-vs \
    --num-requests=1 \
    --app=httpbin
```

Expected:

- Upstream metrics with `responseCode=0` increase by `num-requests * (timeout / perTryTimeout)`
- All downstream responses are HTTP 504 (Gateway Timeout)

## Resilience Executor

The resilience executor is a comprehensive tool that parses and executes resilience test cases from a JSON configuration file. It's designed to sequentially run multiple test scenarios with different assertion types, supporting automatic metric validation with retry logic.

### Building the Executor

```bash
go build -o resilience-executor ./cmd/test-runner
```

### Test File Format

The executor uses a JSON file to define test cases. Each test case includes:

- **name**: Test case identifier
- **description**: Human-readable description of what the test validates
- **type**: Test category (retry-5xx, retry-tcp-errors, timeout, ok-response, outlier-detection)
- **prometheus_url**: Prometheus server endpoint
- **service_endpoint**: Target service URL for traffic generation
- **namespace**: Kubernetes namespace
- **virtual_service**: Name of the VirtualService resource (optional if using destination_rule)
- **destination_rule**: Name of the DestinationRule resource (optional if using virtual_service)
- **num_requests**: Number of requests to generate
- **expected_response_codes**: Array of expected HTTP status codes (validated during execution)
- **app**: Application/destination name for metrics queries
- **assertions**: Array of metric assertions to validate

Each assertion includes:

- **source**: Currently supports "metric"
- **metric_name**: Name of the Prometheus metric (e.g., istio_requests_total, istio_tcp_connections_closed_total)
- **labels**: Label selectors for the metric query
- **expected**: Object with operator and value to validate against
  - **operator**: Comparison operator (eq, gte, lte, gt, lt)
  - **value**: Expected metric difference value

### Example Test Configuration

See [resilience-tests.json](resilience-tests.json) for complete examples. Here's a sample:

```json
{
    "name": "upstream5xxFailures",
    "description": "Test to verify gateway retries on upstream 5xx failures",
    "prometheus_url": "http://prometheus.local",
    "service_endpoint": "http://httpbin.local/status/500",
    "namespace": "demo",
    "virtual_service": "httpbin-vs",
    "num_requests": 1,
    "expected_response_codes": [500],
    "app": "httpbin",
    "type": "retry-5xx",
    "assertions": [
        {
            "source": "metric",
            "metric_name": "istio_requests_total",
            "labels": {
                "destination_app": "httpbin",
                "response_code": "500",
                "namespace": "demo"
            },
            "expected": {
                "operator": "eq",
                "value": 4
            }
        }
    ]
}
```

### Running Tests

#### Using the Shell Script (Recommended)

The script automatically builds the binary before running tests:

```bash
./scripts/resilience-executor.sh
```

With options:

```bash
./scripts/resilience-executor.sh --test upstream5xxFailures --delay 3 --retries 10
./scripts/resilience-executor.sh -f custom-tests.json -t outlierDetectionVerify
```

#### Using the Binary Directly

Build first:

```bash
go build -o resilience-executor ./cmd/test-runner
```

Then run:

```bash
./resilience-executor --test-file=resilience-tests.json
```

#### Run a Specific Test

```bash
./scripts/resilience-executor.sh --test upstream5xxFailures
```

Or directly with binary:

```bash
./resilience-executor --test-file=resilience-tests.json --test-name=upstream5xxFailures
```

#### Custom Retry Configuration

```bash
./scripts/resilience-executor.sh --retries 10 --delay 3
```

Or with binary:

```bash
./resilience-executor \
    --test-file=resilience-tests.json \
    --max-retries=10 \
    --delay-seconds=3
```

### Test Execution Flow

For each test case, the executor:

1. **Validates Configuration**: Ensures all required fields are present
2. **Captures Baseline Metrics**: Queries Prometheus for initial metric values
3. **Generates Traffic**: Sends the specified number of requests to the service
4. **Validates Response Codes**: Verifies actual response codes match expected values
5. **Waits for Metrics Sync**: Implements retry logic (default: 5 retries with 5-second delays)
6. **Queries Metrics After**: Fetches updated metric values from Prometheus
7. **Validates Assertions**: Checks if metric differences match expected values using comparison operators
8. **Reports Results**: Displays PASSED/FAILED status with detailed assertion breakdown

### Assertion Operators

- **eq**: Exact equality (actual == expected)
- **gte**: Greater than or equal (actual >= expected)
- **lte**: Less than or equal (actual <= expected)
- **gt**: Greater than (actual > expected)
- **lt**: Less than (actual < expected)

### Output Features

The executor provides:

- **Welcome Banner**: Clear test suite identification
- **Test Counter**: Shows [N/M] for test progress
- **Real-time Logging**: Detailed step-by-step execution logs
- **Response Code Validation**: Visual indicators (✓/✗) for each request
- **Assertion Tracking**: Per-assertion pass/fail status with metric deltas
- **Progress Indicators**: Visual feedback during metric syncing
- **Summary Report**: 
  - Test and assertion pass rates
  - Progress bar showing completion percentage
  - Color-coded final status
  - Clear success/failure banners

Example output:

```
╔═══════════════════════════════════════════════════════════════╗
║           OkResilience - Gateway Resilience Validator         ║
╚═══════════════════════════════════════════════════════════════╝

[1/5] Executing: upstream5xxFailures (retry-5xx)
  ⏳ Capturing baseline metrics (1 assertion(s))...
    • Assertion 0: istio_requests_total = 38
  ✓ Baseline metrics captured
  🚀 Generating 1 request(s)...
    • Request 1: ✓ 500
  ✓ All response codes matched expectations
  ⏳ Validating metrics (max 5 retries, 5 second delays)...
    • Assertion 0: ✓ (diff=4, expected eq 4)
  ✓ Metrics validation passed on attempt 1

┌─────────────────────────────────────────────────────────────────
  ✓ PASSED | upstream5xxFailures
  Assertions: 1 passed, 0 failed (total: 1)
─────────────────────────────────────────────────────────────────

╔═══════════════════════════════════════════════════════════════╗
║                    📊 EXECUTION SUMMARY                       ║
║  ✓ All tests PASSED! Gateway resilience verified successfully. ║
╚═══════════════════════════════════════════════════════════════╝
```

### Command-Line Flags

- `--test-file` (default: "resilience-tests.json"): Path to the test cases JSON file
- `--max-retries` (default: 5): Maximum number of retries for metric validation
- `--delay-seconds` (default: 5): Delay in seconds between retry attempts
- `--test-name` (default: ""): Optional filter to run a specific test by name

## Running Unit Tests

```bash
go test ./... -v
```

Test the resilience executor specifically:

```bash
go test ./cmd/test-runner -v
```

## TCP Failure Simulation

Deploy the `tcp-reset-service` for testing TCP failures:

```bash
./scripts/deploy-tcp-reset.sh
```
