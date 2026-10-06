# Application Template

Base template for generated applications deployed through the demo system.

## Structure

```bash
demo-app-template/
├── main.go              # Go application source
├── Dockerfile           # Multi-stage Docker build
├── .github/
│   └── workflows/
│       └── ci.yml       # GitHub Actions CI/CD pipeline
└── kubernetes/
    ├── 01-namespace.yaml        # Namespace & service account
    ├── 02-deployment.yaml       # Deployment with health checks
    ├── 03-service.yaml          # ClusterIP service
    └── 04-gateway-and-route.yaml # Cilium gateway & HTTPRoute
```

## Application

### What It Does

Minimal web application that displays:

- "Hello {PERSON_NAME} from Kubernetes"
- Pod hostname and IP
- Deployment timestamp
- Health status

### Endpoints

- `GET /` - HTML page with greeting
- `GET /health` - Health check (JSON)
- `GET /metrics` - Prometheus metrics

### Environment Variables

- `PERSON_NAME` - Name to display (default: "World")
- `PORT` - Server port (default: 8080)
- `POD_NAME` - Injected by Kubernetes
- `POD_NAMESPACE` - Injected by Kubernetes
- `POD_IP` - Injected by Kubernetes

## CI/CD Pipeline

### GitHub Actions Workflow (`.github/workflows/ci.yml`)

**Triggers**: Push to main branch

**Jobs**:

1. **Test**
   - Go vet (static analysis)
   - Format check (gofmt)
   - Linting (golangci-lint)
   - Build verification
   - Coverage upload

2. **Build and Push** (runs if tests pass)
   - Docker multi-stage build
   - Push to registry with tags:
     - `:latest`
     - `:{GIT_SHA}`

**Required Secrets** (in GitHub repository):

- `DOCKER_REGISTRY` - Registry URL
- `DOCKER_USERNAME` - Registry username
- `DOCKER_PASSWORD` - Registry password/token

### Build Process

```dockerfile
# Stage 1: Builder
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go build -o app main.go

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/app .
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3
CMD ["./app"]
```

**Image Size**: ~15MB (optimized alpine)

## Kubernetes Manifests

### 1. Namespace (`01-namespace.yaml`)

- Creates `demo-apps` namespace
- Service account for pods

### 2. Deployment (`02-deployment.yaml`)

**Configuration**:

- 2 replicas for high availability
- Rolling update strategy
- Resource requests: 50m CPU, 64Mi memory
- Resource limits: 200m CPU, 256Mi memory
- Pod anti-affinity (spread across nodes)

**Health Checks**:

- Liveness probe: `/health` endpoint (10s interval)
- Readiness probe: `/health` endpoint (5s interval)
- Grace period: 30s

**Template Variables**:

- `{{ APP_NAME }}` - Deployment name
- `{{ PERSON_NAME }}` - Name to display
- `{{ DOCKER_IMAGE }}` - Image URL
- `{{ CREATED_AT }}` - Creation timestamp
- `{{ CLEANUP_TIME }}` - Cleanup deadline

### 3. Service (`03-service.yaml`)

- Type: ClusterIP (internal only)
- Port 80 → 8080
- Labels for discovery

### 4. Gateway & HTTPRoute (`04-gateway-and-route.yaml`)

**Gateway**:

- Gateway class: cilium
- Listeners: HTTP (80) + HTTPS (443)
- Handles all demo apps

**HTTPRoute**:

- Hostnames: `{APP_NAME}.demo.local`, `{APP_NAME}.{DOMAIN}`
- Path: `/` (catch-all)
- Backend: Service:80
- Request header filters

### 5. Monitoring (`05-monitoring.yaml`)

**ServiceMonitor**:

- Scrapes `/metrics` endpoint
- 30s interval
- Prometheus job label

**ConfigMap (Grafana Dashboard)**:

- JSON dashboard definition
- Panels:
  - Request rate
  - Latency percentiles
  - Success rate
  - Uptime
  - Pod availability
- Auto-provisioned via label

## Customization

### Change Application Language

To use Python instead of Go:

```python
# main.py
from flask import Flask, jsonify
app = Flask(__name__)

@app.route('/')
def hello():
    name = os.environ.get('PERSON_NAME', 'World')
    return f'Hello {name} from Kubernetes'

@app.route('/health')
def health():
    return jsonify({'status': 'healthy'})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

```dockerfile
FROM python:3.11-alpine
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY main.py .
CMD ["python", "main.py"]
```

### Metrics

The demo application exposes real-time Prometheus metrics on the `/metrics` endpoint.

The application now exports the following custom metric families:

- `app_requests_total{method, path, status}` - count of HTTP requests by method, path, and response status
- `app_request_duration_seconds_bucket`, `app_request_duration_seconds_sum`, `app_request_duration_seconds_count` - request latency histogram
- `app_requests_in_flight` - currently active requests
- `app_uptime_seconds` - application uptime in seconds
- `app_go_routines` - number of active goroutines
- `app_memory_alloc_bytes` - current bytes allocated by the app

In addition, standard Go and process metrics are exposed automatically by Prometheus client collectors.

Prometheus can scrape the metrics endpoint directly from the pod using `/metrics`.

### Increase Replicas

Edit `kubernetes/02-deployment.yaml`:

```yaml
spec:
  replicas: 3  # Change from 2
```

### Adjust Resource Limits

Edit `kubernetes/02-deployment.yaml`:

```yaml
resources:
  requests:
    cpu: 100m       # Increase from 50m
    memory: 128Mi   # Increase from 64Mi
  limits:
    cpu: 500m       # Increase from 200m
    memory: 512Mi   # Increase from 256Mi
```

### Add Environment Variables

Edit `kubernetes/02-deployment.yaml`:

```yaml
env:
- name: MY_VAR
  value: "my-value"
- name: ANOTHER_VAR
  valueFrom:
    configMapKeyRef:
      name: my-config
      key: my-key
```

### Enable TLS

1. Add certificate to gateway
2. Update HTTPRoute with TLS

```yaml
listeners:
- name: https
  port: 443
  protocol: HTTPS
  tls:
    mode: Terminate
    certificateRefs:
    - name: tls-cert
```

## Testing

### Local Development

```bash
git clone https://github.com/diablinux/demo-app-template
cd demo-app-template

# Build
go build -o app main.go

# Run
./app

# Test
curl http://localhost:8080
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

### Docker Testing

```bash
# Build image
docker build -t my-app:latest .

# Run container
docker run -p 8080:8080 \
  -e PERSON_NAME="Test User" \
  my-app:latest

# Test
curl http://localhost:8080
```

### Kubernetes Testing

```bash
# Deploy
kubectl apply -f kubernetes/

# Port forward
kubectl -n demo-apps port-forward deployment/my-app 8080:8080

# Test
curl http://localhost:8080

# Check metrics
curl http://localhost:8080/metrics

# Check logs
kubectl -n demo-apps logs deployment/my-app

# Describe deployment
kubectl -n demo-apps describe deployment my-app
```

## Performance

### Response Time

- Cold start: ~100ms
- Health check: ~5ms
- Metrics endpoint: ~20ms

### Resource Usage

- Per pod: ~5-10MB memory
- CPU: Minimal (mostly idle)
- Network: Negligible

## Monitoring

### Prometheus Queries

```prometheus
# Request rate
rate(app_requests_total[5m])

# Error rate
rate(app_requests_total{status=~"5.."}[5m])

# Latency (p95)
histogram_quantile(0.95, rate(app_request_duration_seconds_bucket[5m]))

# Uptime (pod running)
up{job="my-app"}
```

### Grafana Dashboard

The dashboard is auto-provisioned from ConfigMap:

- Request metrics
- Latency percentiles
- Success rate
- Pod availability
- Service status

## Troubleshooting

### Pod won't start

```bash
# Check pod status
kubectl -n demo-apps describe pod my-app-xxxxx

# Check logs
kubectl -n demo-apps logs my-app-xxxxx

# Check events
kubectl -n demo-apps get events
```

### Application not responding

```bash
# Check service
kubectl -n demo-apps get svc

# Test pod directly
kubectl -n demo-apps port-forward pod/my-app-xxxxx 8080:8080
curl http://localhost:8080

# Check gateway
kubectl -n demo-apps describe gateway demo-gateway
```

### High latency

```bash
# Check resources
kubectl -n demo-apps top pods

# Check node load
kubectl top nodes

# Increase CPU/memory limits
kubectl -n demo-apps set resources deployment/my-app \
  --limits=cpu=500m,memory=512Mi
```

## Production Checklist

- [ ] Use specific image tags (not :latest)
- [ ] Set resource limits appropriately
- [ ] Enable security context (non-root user)
- [ ] Add network policies
- [ ] Configure RBAC properly
- [ ] Enable pod security policies
- [ ] Set up log aggregation
- [ ] Configure alerting
- [ ] Document configuration
- [ ] Test disaster recovery
- [ ] Plan capacity
- [ ] Monitor costs

## References

- [Go Standard Library](https://golang.org/pkg/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Kubernetes Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Prometheus Metrics](https://prometheus.io/docs/practices/instrumentation/)
