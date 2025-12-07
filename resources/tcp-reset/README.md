# TCP Reset Service

A minimal Go TCP server that immediately closes connections with `SO_LINGER=0` to emit TCP RST packets, simulating downstream TCP failures for resilience testing.

## Overview

This service listens on port 80 and forcibly closes every incoming connection, causing clients to observe:

- Connection reset by peer
- Empty reply from server
- Abrupt connection failures

This is useful for testing:

- Application resilience to TCP-level failures
- Circuit breakers and retry logic
- Client-side error handling
- Connection timeout behaviors

## Quick Start

### Build and Deploy

```zsh
# Build the Docker image
docker build -t tcp-reset-service:local resources/tcp-reset

# If using kind, load into cluster
kind load docker-image tcp-reset-service:local --name ok-resilience

# Deploy to Kubernetes
kubectl apply -f resources/tcp-reset/deployment.yaml

# Verify pod is running
kubectl -n tcp-ns get pods -l app=tcp-reset-service
```

### Route Traffic

To test with httpbin, apply the VirtualService to route requests to this failing service:

```zsh
kubectl apply -f tcp-reset-virtualservice.yaml
```

### Test

```zsh
# Expect connection failures
for i in {1..10}; do 
  curl -v --max-time 3 http://httpbin.local/status/200 || echo "Connection failed"
done
```

## Observability

**Important**: TCP-level failures occur before HTTP processing, so:

- ❌ **Won't appear in**: `istio_requests_total` (no HTTP request is formed)
- ✅ **Will appear in**:
  - Client-side connection errors
  - `istio_tcp_connections_closed_total` metrics
  - Source-side Envoy access logs with flags like `UC`, `UF`, `URX`
  - Application error logs and retry counters

### Where to Look

- **Client side**: Check logs and metrics from the service making requests
- **TCP metrics**: `istio_tcp_*` metrics from source workload
- **Access logs**: Enable with `istioctl install --set meshConfig.accessLogFile="/dev/stdout"`

## Cleanup

```zsh
# Remove VirtualService
kubectl delete -f tcp-reset-virtualservice.yaml

# Remove deployment
kubectl delete -f deployment.yaml
```

## How It Works

The server uses Go's `syscall` package to set `SO_LINGER` with a timeout of 0 before closing connections:

```go
linger := &syscall.Linger{Onoff: 1, Linger: 0}
syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, linger)
conn.Close() // Emits TCP RST
```

This forces the TCP stack to send a RST packet instead of a graceful FIN, simulating abrupt connection failures.

## Use Cases

- Test application resilience to network failures
- Validate retry and circuit breaker logic
- Stress-test connection pool behavior
- Verify timeout configurations
- Simulate upstream service crashes
