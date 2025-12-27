#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Color codes
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print message with yellow tick
print_msg() {
  echo -e "${YELLOW}$1 ${NC}"
}

print_msg "Running resilience test for upstream 5xx failures..."
./okresilience upstream5xxFailures \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://httpbin.local/status/500 \
    --namespace=demo \
    --virtual-service=httpbin-vs \
    --num-requests=1 \
    --response-code=500 \
    --app=httpbin

print_msg "Running resilience test for upstream TCP resets..."
./okresilience upstreamTcpReset \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://tcp-reset-service.local/ \
    --namespace=tcp-ns \
    --virtual-service=tcp-reset-vs \
    --num-requests=1 \
    --response-code=503 \
    --app=tcp-reset-service 

print_msg "Running resilience test for gateway timeout verification..."
./okresilience gatewayTimeoutVerify \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://httpbin.local/delay/3 \
    --namespace=demo \
    --virtual-service=httpbin-vs \
    --num-requests=1 \
    --app=httpbin


print_msg "Generating traffic to httpbin service to reset upstream stats in ingress gateway..."
./scripts/generate-traffic.sh status 10

print_msg "Running resilience test for gateway outlier detection verification..."
./okresilience outlierDetectionVerify \
    --prometheus-url=http://prometheus.local \
    --service-endpoint=http://httpbin.local/status/500 \
    --namespace=demo \
    --virtual-service=httpbin-vs \
    --destination-rule=httpbin-dr \
    --num-requests=5 \
    --app=httpbin

# After outlier detection test, wait for some time to allow Envoy to recover
# Upstream stats may take some time to reset after outlier ejection
sleep 90