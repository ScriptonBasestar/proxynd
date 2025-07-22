# ProxyND Deployment Guide

This document provides comprehensive guidance for deploying ProxyND in various environments, from development to production.

## Overview

ProxyND supports multiple deployment strategies:
- **Manual Deployment**: Using scripts and manual processes
- **Automated Deployment**: Using GitHub Actions and CI/CD pipelines
- **Container Deployment**: Using Docker and Kubernetes
- **Helm Deployment**: Using Kubernetes Helm charts

## Prerequisites

### System Requirements

**Minimum Requirements:**
- CPU: 2 cores
- Memory: 4GB RAM
- Disk: 20GB available space
- Network: Stable internet connection

**Recommended Requirements:**
- CPU: 4+ cores
- Memory: 8GB+ RAM
- Disk: 100GB+ available space (for cache storage)
- Network: High-bandwidth connection

### Software Requirements

- **Go**: 1.21+ (for building from source)
- **Docker**: 20.10+ (for container deployment)
- **Kubernetes**: 1.24+ (for Kubernetes deployment)
- **Helm**: 3.10+ (for Helm deployment)
- **kubectl**: Compatible with your Kubernetes version

## Deployment Methods

### 1. Binary Deployment

#### Download Pre-built Binaries

```bash
# Download latest release
curl -LO https://github.com/scriptonbasestar/proxynd/releases/latest/download/proxynd-linux-amd64

# Make executable
chmod +x proxynd-linux-amd64
sudo mv proxynd-linux-amd64 /usr/local/bin/proxynd

# Verify installation
proxynd --version
```

#### Build from Source

```bash
# Clone repository
git clone https://github.com/scriptonbasestar/proxynd.git
cd proxynd

# Build binary
make build

# Install
sudo cp bin/proxynd /usr/local/bin/
```

#### Configuration

```bash
# Create configuration directory
sudo mkdir -p /etc/proxynd

# Copy sample configuration
sudo cp sample-conf/* /etc/proxynd/

# Edit configuration
sudo nano /etc/proxynd/global.yaml
```

#### Running as Service

Create systemd service file:

```bash
sudo tee /etc/systemd/system/proxynd.service << EOF
[Unit]
Description=ProxyND Package Proxy Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=proxynd
Group=proxynd
ExecStart=/usr/local/bin/proxynd --config-dir=/etc/proxynd
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Create user
sudo useradd -r -s /bin/false proxynd

# Set permissions
sudo chown -R proxynd:proxynd /etc/proxynd
sudo mkdir -p /var/lib/proxynd
sudo chown -R proxynd:proxynd /var/lib/proxynd

# Enable and start service
sudo systemctl enable proxynd
sudo systemctl start proxynd
sudo systemctl status proxynd
```

### 2. Docker Deployment

#### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  proxynd:
    image: ghcr.io/scriptonbasestar/proxynd:latest
    container_name: proxynd
    ports:
      - "8080:8080"
    volumes:
      - ./config:/config
      - ./storage:/storage
      - ./logs:/logs
    environment:
      - CONFIG_DIR=/config
      - STORAGE_DIR=/storage
      - LOG_LEVEL=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

  # Optional: Monitoring with Prometheus
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
    restart: unless-stopped

volumes:
  prometheus_data:
```

#### Run with Docker Compose

```bash
# Create directories
mkdir -p config storage logs monitoring

# Copy configuration
cp sample-conf/* config/

# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f proxynd
```

### 3. Kubernetes Deployment

#### Using Raw Manifests

Create `proxynd-namespace.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: proxynd
```

Create `proxynd-configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: proxynd-config
  namespace: proxynd
data:
  global.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"

    cache:
      enabled: true
      directory: "/storage/cache"
      max_size: "10GB"

    logging:
      level: "info"
      format: "json"

    monitoring:
      enabled: true
      prometheus:
        enabled: true
        path: "/metrics"

  apt-proxy.yaml: |
    apt-proxy:
      enabled: true
      upstream_url: "http://archive.ubuntu.com/ubuntu/"
      cache_dir: "/storage/apt-cache"

  maven-proxy.yaml: |
    maven-proxy:
      enabled: true
      upstream_url: "https://repo1.maven.org/maven2/"
      cache_dir: "/storage/maven-cache"
```

Create `proxynd-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxynd
  namespace: proxynd
  labels:
    app: proxynd
spec:
  replicas: 3
  selector:
    matchLabels:
      app: proxynd
  template:
    metadata:
      labels:
        app: proxynd
    spec:
      containers:
      - name: proxynd
        image: ghcr.io/scriptonbasestar/proxynd:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: CONFIG_DIR
          value: "/config"
        - name: STORAGE_DIR
          value: "/storage"
        volumeMounts:
        - name: config
          mountPath: /config
        - name: storage
          mountPath: /storage
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
      volumes:
      - name: config
        configMap:
          name: proxynd-config
      - name: storage
        persistentVolumeClaim:
          claimName: proxynd-storage
```

Create `proxynd-pvc.yaml`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: proxynd-storage
  namespace: proxynd
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: fast-ssd
```

Create `proxynd-service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: proxynd
  namespace: proxynd
  labels:
    app: proxynd
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: proxynd
```

Create `proxynd-ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: proxynd
  namespace: proxynd
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
spec:
  tls:
  - hosts:
    - proxynd.example.com
    secretName: proxynd-tls
  rules:
  - host: proxynd.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: proxynd
            port:
              number: 8080
```

Deploy to Kubernetes:

```bash
# Apply manifests
kubectl apply -f proxynd-namespace.yaml
kubectl apply -f proxynd-configmap.yaml
kubectl apply -f proxynd-pvc.yaml
kubectl apply -f proxynd-deployment.yaml
kubectl apply -f proxynd-service.yaml
kubectl apply -f proxynd-ingress.yaml

# Check deployment
kubectl get pods -n proxynd
kubectl get svc -n proxynd
kubectl get ingress -n proxynd
```

### 4. Helm Deployment

#### Quick Start

```bash
# Add Helm repository (when available)
helm repo add proxynd https://charts.proxynd.io
helm repo update

# Install with default values
helm install proxynd proxynd/proxynd

# Install with custom values
helm install proxynd proxynd/proxynd -f values.yaml
```

#### Using Local Helm Chart

```bash
# Clone repository
git clone https://github.com/scriptonbasestar/proxynd.git
cd proxynd

# Install using local chart
helm install proxynd ./helm --namespace proxynd --create-namespace

# Upgrade
helm upgrade proxynd ./helm --namespace proxynd
```

#### Custom Values Example

Create `values.yaml`:

```yaml
image:
  repository: ghcr.io/scriptonbasestar/proxynd
  tag: "latest"
  pullPolicy: IfNotPresent

replicaCount: 3

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: proxynd.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: proxynd-tls
      hosts:
        - proxynd.example.com

resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

persistence:
  enabled: true
  storageClass: "fast-ssd"
  size: 100Gi

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true

config:
  global:
    server:
      port: 8080
    cache:
      max_size: "50GB"
    performance:
      enabled: true

  proxies:
    apt:
      enabled: true
      upstream_url: "http://archive.ubuntu.com/ubuntu/"
    maven:
      enabled: true
      upstream_url: "https://repo1.maven.org/maven2/"
    npm:
      enabled: true
      upstream_url: "https://registry.npmjs.org/"
```

Deploy with custom values:

```bash
helm install proxynd ./helm -f values.yaml --namespace proxynd --create-namespace
```

## Automated Deployment

### GitHub Actions Release Workflow

The repository includes a comprehensive GitHub Actions workflow for automated releases and deployments. The workflow triggers on:

1. **Tag Push**: When a version tag (e.g., `v1.0.0`) is pushed
2. **Manual Trigger**: Using workflow dispatch with custom parameters

#### Workflow Features

- **Multi-platform Builds**: Supports Linux, macOS, and Windows
- **Security Scanning**: Automated vulnerability scanning
- **Container Images**: Multi-architecture Docker images
- **Helm Charts**: Automated chart packaging and publishing
- **Deployment**: Automated deployment to staging and production
- **Notifications**: Success/failure notifications
- **Rollback**: Automatic rollback on deployment failures

#### Triggering a Release

```bash
# Tag-based release
git tag v1.0.0
git push origin v1.0.0

# Manual release via GitHub UI
# Go to Actions → Release → Run workflow
```

### Environment Setup

#### Staging Environment

Configure staging environment secrets:

```bash
# Kubernetes config
STAGING_KUBECONFIG: <base64-encoded-kubeconfig>

# Optional: Container registry credentials
DOCKERHUB_USERNAME: <username>
DOCKERHUB_TOKEN: <token>

# Optional: Notification webhooks
SLACK_WEBHOOK_URL: <webhook-url>
```

#### Production Environment

Configure production environment secrets with appropriate protection rules:

```bash
# Kubernetes config
PRODUCTION_KUBECONFIG: <base64-encoded-kubeconfig>

# Container signing (optional)
COSIGN_KEY: <cosign-private-key>

# Monitoring/alerting
MONITORING_WEBHOOK: <webhook-url>
```

### Manual Deployment Scripts

#### Using Deployment Script

```bash
# Deploy to staging
./scripts/deploy.sh -e staging -v v1.0.0

# Deploy to production
./scripts/deploy.sh -e production -v v1.0.0

# Dry run
./scripts/deploy.sh -e staging -v v1.0.0 --dry-run

# Rollback
./scripts/deploy.sh -e staging --rollback
```

#### Running Smoke Tests

```bash
# Test staging deployment
./scripts/smoke-tests.sh https://staging.proxynd.example.com

# Test production deployment
./scripts/smoke-tests.sh https://proxynd.example.com

# Verbose testing
./scripts/smoke-tests.sh -v https://proxynd.example.com

# Test specific proxy types
./scripts/smoke-tests.sh -p apt,maven https://proxynd.example.com
```

## Monitoring and Observability

### Health Checks

ProxyND provides several health check endpoints:

- `/health` - Overall health status
- `/health/ready` - Readiness check
- `/health/live` - Liveness check

### Metrics

Prometheus metrics are available at `/metrics` endpoint:

- Request counters and durations
- Cache hit/miss ratios
- System resource usage
- Error rates and types

### Logging

Structured JSON logging with configurable levels:

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, file
```

### Monitoring Stack

Deploy monitoring stack with ProxyND:

```bash
# Using provided monitoring configuration
kubectl apply -f monitoring/

# Using Helm
helm install monitoring ./monitoring/helm
```

Includes:
- **Prometheus**: Metrics collection
- **Grafana**: Dashboards and visualization
- **AlertManager**: Alert routing and notifications
- **Loki**: Log aggregation
- **Jaeger**: Distributed tracing

## Troubleshooting

### Common Issues

#### Pod Startup Issues

```bash
# Check pod status
kubectl get pods -n proxynd

# Check pod logs
kubectl logs -f deployment/proxynd -n proxynd

# Describe pod for events
kubectl describe pod <pod-name> -n proxynd
```

#### Storage Issues

```bash
# Check PVC status
kubectl get pvc -n proxynd

# Check storage class
kubectl get storageclass

# Check persistent volume
kubectl get pv
```

#### Network Issues

```bash
# Check service
kubectl get svc -n proxynd

# Check ingress
kubectl get ingress -n proxynd

# Test service connectivity
kubectl port-forward svc/proxynd 8080:8080 -n proxynd
curl http://localhost:8080/health
```

#### Configuration Issues

```bash
# Check configmap
kubectl get configmap proxynd-config -n proxynd -o yaml

# Update configuration
kubectl edit configmap proxynd-config -n proxynd

# Restart deployment
kubectl rollout restart deployment/proxynd -n proxynd
```

### Debug Mode

Enable debug logging:

```yaml
logging:
  level: "debug"
```

Or set environment variable:

```bash
export LOG_LEVEL=debug
```

### Performance Issues

Check resource usage:

```bash
# CPU and memory usage
kubectl top pods -n proxynd

# Resource limits
kubectl describe pod <pod-name> -n proxynd

# Check metrics
curl http://localhost:8080/metrics
```

## Security Considerations

### Container Security

- Use non-root user in containers
- Scan images for vulnerabilities
- Use minimal base images
- Set resource limits
- Enable security contexts

### Network Security

- Use network policies
- Enable TLS/HTTPS
- Restrict ingress rules
- Use service mesh (optional)

### Access Control

- Configure RBAC
- Use service accounts
- Implement authentication
- Set up audit logging

### Secrets Management

- Use Kubernetes secrets
- External secret managers
- Regular secret rotation
- Principle of least privilege

## Best Practices

### High Availability

- Deploy multiple replicas
- Use pod disruption budgets
- Implement health checks
- Configure autoscaling
- Use multiple availability zones

### Scalability

- Enable horizontal pod autoscaling
- Use persistent storage
- Configure resource requests/limits
- Monitor performance metrics
- Plan for traffic growth

### Backup and Recovery

- Regular configuration backups
- Storage volume snapshots
- Database backups (if applicable)
- Disaster recovery procedures
- Test recovery processes

### Updates and Maintenance

- Use rolling updates
- Test in staging first
- Monitor during deployments
- Have rollback procedures
- Schedule maintenance windows

## Production Checklist

### Pre-deployment

- [ ] Resource requirements validated
- [ ] Configuration reviewed and tested
- [ ] Security settings configured
- [ ] Monitoring and alerting set up
- [ ] Backup procedures in place
- [ ] Rollback procedures tested

### Deployment

- [ ] Staging deployment successful
- [ ] Smoke tests passed
- [ ] Performance tests completed
- [ ] Security scans passed
- [ ] Documentation updated

### Post-deployment

- [ ] Health checks passing
- [ ] Metrics being collected
- [ ] Logs being aggregated
- [ ] Performance monitoring active
- [ ] Alerts configured
- [ ] Team notified

For additional support and documentation, visit the [ProxyND GitHub repository](https://github.com/scriptonbasestar/proxynd) or consult the [troubleshooting guide](troubleshooting.md).
