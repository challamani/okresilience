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
brew install kind
kind --version
#start the docker instance 
#create the test cluster
kind create cluster --name ok-resilience
kind get clusters
#verify current context
kubectl config get-contexts
```

## Install istio

```shell
istioctl install --set profile=demo -y
```
## Kind LoadBalancer service 

- [Load Balancer](https://kind.sigs.k8s.io/docs/user/loadbalancer/) service.

```shell
#install cloud-provider-kind
brew install cloud-provider-kind

#new tab - run the following
cloud-provider-kind --gateway-channel standard
```

## Deploy a sample 

```shell
kubectl apply -f okresilience/resources/httpbin.yaml

#verify the external-ip of istio-ingress gateway
kubectl get service -n istio-system -o wide
#add hostname mapping in /etc/hosts
#<external-ip> httpbin.local

curl -v http://httpbin.local/status/200
```

## Install Kiali Dashboard

[Kiali Resources](https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/kiali.yaml)

```shell
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/kiali.yaml
#verify that kiali resources installed in istio-system namespace
```

[Prometheus Resources](https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/prometheus.yaml)
```shell
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.28/samples/addons/prometheus.yaml
#verify that prometheus resources installed in istio-system namespace

#Access kiali by creating gateway and virtual-service at istio-ingressgateway
kubectl apply -f resources/kiali-gateway.yaml 
#add hostname mapping in /etc/host 
#<external-ip> kiali.local
```

http://kiali.local/
