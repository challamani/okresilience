#!/bin/bash

set -euo pipefail

MODE=${1:-status}
REQUEST_COUNT=${REQUEST_COUNT:-50}
INGRESS_IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [[ -z "${INGRESS_IP}" ]]; then
    echo "Could not determine istio-ingressgateway external IP. Ensure the gateway service is provisioned."
    exit 1
fi

case "${MODE}" in
    status)
        ENDPOINT="/status/200"
        CURL_FLAGS=(-s -D - -o /dev/null)
        ;;
    header)
        ENDPOINT="/get"
        CURL_FLAGS=(-s)
        ;;
    delay)
        ENDPOINT="/delay/5"
        CURL_FLAGS=(-s -D - -o /dev/null)
        ;;
    *)
        echo "Usage: $0 [status|header|delay]"
        exit 1
        ;;
esac

for ((i=1; i<=REQUEST_COUNT; i++)); do
    echo -e "\nRequest ==> [${i}], endpoint=${ENDPOINT}"
    curl "${CURL_FLAGS[@]}" -H "Host: httpbin.local" "http://${INGRESS_IP}${ENDPOINT}"
    sleep 1
done