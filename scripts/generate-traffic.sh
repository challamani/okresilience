#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e
export INGRESS_IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

for i in {1..31}; 
    do echo -e "\nRequest ==> [$i]"; 
    curl -s -D - -o /dev/null -H "Host: httpbin.local" http://$INGRESS_IP/status/200;
    #curl -H "Host: httpbin.local" http://$INGRESS_IP/get; 
    sleep 1;
done