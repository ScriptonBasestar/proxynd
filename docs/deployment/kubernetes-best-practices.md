# Kubernetes Deployment Best Practices

Best practices for deploying ProxyND Enterprise API on Kubernetes.

## 📋 Overview

This guide covers production-ready Kubernetes deployment patterns, resource management, auto-scaling, and operational best practices.

**Target Audience**: DevOps engineers, SREs, Platform engineers
**Kubernetes Version**: 1.25+ (tested up to 1.29)

---

## 🏗️ Architecture Overview

### Recommended Topology

```
┌─────────────────────────────────────────────────────────┐
│                    Ingress Controller                    │
│            (nginx-ingress / ALB / Traefik)              │
└─────────────────┬───────────────────────────────────────┘
                  │
         ┌────────┴────────┐
         │    Service      │
         │  (LoadBalancer) │
         └────────┬────────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
┌───▼────┐   ┌───▼────┐   ┌───▼────┐
│  Pod   │   │  Pod   │   │  Pod   │
│ProxyND │   │ProxyND │   │ProxyND │
└───┬────┘   └───┬────┘   └───┬────┘
    │            │            │
    └────────────┴────────────┘
                 │
        ┌────────┴────────┐
        │                 │
    ┌───▼────┐      ┌────▼────┐
    │ Redis  │      │Database │
    │ (Cache)│      │(Postgres│
    └────────┘      └─────────┘
```

### Components

- **Deployment**: ProxyND application pods
- **Service**: Load balancing across pods
- **Ingress**: External HTTPS access
- **ConfigMap**: Configuration files
- **Secret**: Sensitive credentials
- **PersistentVolumeClaim**: Cache storage
- **HorizontalPodAutoscaler**: Auto-scaling based on load
- **PodDisruptionBudget**: Ensure availability during updates

---

## 📦 Resource Configuration

### 1. Deployment Manifest

**File**: `k8s/proxynd-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxynd
  namespace: production
  labels:
    app: proxynd
    component: enterprise-api
    version: v1.0.0
spec:
  replicas: 3  # Start with 3 for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0  # Zero-downtime deployments
  selector:
    matchLabels:
      app: proxynd
  template:
    metadata:
      labels:
        app: proxynd
        component: enterprise-api
        version: v1.0.0
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      # Security context
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault

      # Init container for config validation
      initContainers:
        - name: config-validator
          image: proxynd:v1.0.0
          command: ["/usr/local/bin/proxyndctl"]
          args: ["config", "validate"]
          volumeMounts:
            - name: config
              mountPath: /etc/proxynd
              readOnly: true

      # Application container
      containers:
        - name: proxynd
          image: proxynd:v1.0.0
          imagePullPolicy: IfNotPresent

          # Resource limits (CRITICAL for production)
          resources:
            requests:
              cpu: 500m      # 0.5 CPU
              memory: 1Gi    # 1GB RAM
            limits:
              cpu: 2000m     # 2 CPU
              memory: 4Gi    # 4GB RAM

          # Ports
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP

          # Environment variables
          env:
            - name: GO_ENV
              value: "production"
            - name: CONFIG_DIR
              value: "/etc/proxynd"
            - name: STORAGE_DIR
              value: "/var/lib/proxynd"
            - name: LOG_LEVEL
              value: "info"
            - name: LOG_FORMAT
              value: "json"

          # Liveness probe (restart if unhealthy)
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3

          # Readiness probe (remove from load balancer if not ready)
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 2

          # Startup probe (for slow-starting applications)
          startupProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 0
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 30  # 300s (5min) max startup time

          # Volume mounts
          volumeMounts:
            - name: config
              mountPath: /etc/proxynd
              readOnly: true
            - name: storage
              mountPath: /var/lib/proxynd
            - name: cache
              mountPath: /tmp

          # Security context (container-level)
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            runAsUser: 1000
            capabilities:
              drop:
                - ALL

      # Volumes
      volumes:
        - name: config
          configMap:
            name: proxynd-config
        - name: storage
          persistentVolumeClaim:
            claimName: proxynd-storage
        - name: cache
          emptyDir:
            sizeLimit: 1Gi

      # Affinity rules (spread pods across nodes)
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - proxynd
                topologyKey: kubernetes.io/hostname
```

### 2. Service

**File**: `k8s/proxynd-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: proxynd
  namespace: production
  labels:
    app: proxynd
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"  # AWS NLB
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: "tcp"
spec:
  type: LoadBalancer
  selector:
    app: proxynd
  ports:
    - name: http
      port: 80
      targetPort: 8080
      protocol: TCP
    - name: https
      port: 443
      targetPort: 8080
      protocol: TCP
  sessionAffinity: ClientIP  # Sticky sessions
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 3 hours
```

### 3. Ingress

**File**: `k8s/proxynd-ingress.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: proxynd
  namespace: production
  annotations:
    # nginx-ingress annotations
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"  # Large package uploads
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "30"

    # Rate limiting
    nginx.ingress.kubernetes.io/limit-rps: "100"
    nginx.ingress.kubernetes.io/limit-connections: "10"

    # Certificate (cert-manager)
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
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
                  number: 80
```

### 4. ConfigMap

**File**: `k8s/proxynd-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: proxynd-config
  namespace: production
data:
  config.yaml: |
    server:
      port: 8080
      host: 0.0.0.0
      read_timeout: 30s
      write_timeout: 30s

    storage:
      path: /var/lib/proxynd

    cache:
      backend: redis
      ttl: 24h
      redis:
        host: redis-master.production.svc.cluster.local
        port: 6379
        db: 0

    proxies:
      - type: maven
        enabled: true
        upstream: https://repo1.maven.org/maven2
        cache_ttl: 720h

      - type: npm
        enabled: true
        upstream: https://registry.npmjs.org
        cache_ttl: 168h

    metrics:
      enabled: true
      path: /metrics

    logging:
      level: info
      format: json
```

### 5. Secrets

**File**: `k8s/proxynd-secrets.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: proxynd-secrets
  namespace: production
type: Opaque
stringData:
  # Database credentials
  DB_PASSWORD: "your-db-password"

  # Redis password
  REDIS_PASSWORD: "your-redis-password"

  # OAuth2 secrets
  GITHUB_CLIENT_SECRET: "your-github-secret"
  GOOGLE_CLIENT_SECRET: "your-google-secret"

  # JWT secret
  JWT_SECRET: "your-jwt-secret"

  # S3 credentials (if using S3 cache)
  AWS_ACCESS_KEY_ID: "your-access-key"
  AWS_SECRET_ACCESS_KEY: "your-secret-key"
```

**IMPORTANT**: Never commit secrets to Git. Use:
- External secrets operator
- HashiCorp Vault
- AWS Secrets Manager / GCP Secret Manager
- Sealed Secrets

### 6. PersistentVolumeClaim

**File**: `k8s/proxynd-pvc.yaml`

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: proxynd-storage
  namespace: production
spec:
  accessModes:
    - ReadWriteMany  # Multiple pods can read/write
  resources:
    requests:
      storage: 100Gi
  storageClassName: fast-ssd  # Use SSD for cache performance
```

---

## 📈 Auto-Scaling

### HorizontalPodAutoscaler (HPA)

**File**: `k8s/proxynd-hpa.yaml`

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: proxynd-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: proxynd

  minReplicas: 3    # Minimum for HA
  maxReplicas: 20   # Maximum allowed

  metrics:
    # CPU-based scaling
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70  # Scale up at 70% CPU

    # Memory-based scaling
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80  # Scale up at 80% memory

    # Custom metric: Request rate
    - type: Pods
      pods:
        metric:
          name: proxynd_enterprise_api_requests_per_second
        target:
          type: AverageValue
          averageValue: "500"  # Scale up at 500 rps per pod

  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60  # Wait 60s before scaling up
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60  # Max 50% increase per minute
        - type: Pods
          value: 2
          periodSeconds: 60  # Or max 2 pods per minute
      selectPolicy: Max

    scaleDown:
      stabilizationWindowSeconds: 300  # Wait 5min before scaling down
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60  # Max 10% decrease per minute
        - type: Pods
          value: 1
          periodSeconds: 60  # Or max 1 pod per minute
      selectPolicy: Min
```

### VerticalPodAutoscaler (VPA) - Optional

**File**: `k8s/proxynd-vpa.yaml`

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: proxynd-vpa
  namespace: production
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: proxynd

  updatePolicy:
    updateMode: "Auto"  # Auto-update resource requests

  resourcePolicy:
    containerPolicies:
      - containerName: proxynd
        minAllowed:
          cpu: 500m
          memory: 1Gi
        maxAllowed:
          cpu: 4000m
          memory: 8Gi
        controlledResources: ["cpu", "memory"]
```

---

## 🛡️ High Availability

### PodDisruptionBudget

**File**: `k8s/proxynd-pdb.yaml`

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: proxynd-pdb
  namespace: production
spec:
  minAvailable: 2  # Always keep at least 2 pods running
  selector:
    matchLabels:
      app: proxynd
```

### TopologySpreadConstraints

Add to Deployment spec:

```yaml
spec:
  template:
    spec:
      topologySpreadConstraints:
        # Spread across availability zones
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels:
              app: proxynd

        # Spread across nodes
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app: proxynd
```

---

## 🔒 Security Best Practices

### 1. RBAC (Kubernetes Role-Based Access Control)

**File**: `k8s/proxynd-rbac.yaml`

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: proxynd
  namespace: production

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: proxynd-role
  namespace: production
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: proxynd-rolebinding
  namespace: production
subjects:
  - kind: ServiceAccount
    name: proxynd
    namespace: production
roleRef:
  kind: Role
  name: proxynd-role
  apiGroup: rbac.authorization.k8s.io
```

Add to Deployment:

```yaml
spec:
  template:
    spec:
      serviceAccountName: proxynd
```

### 2. NetworkPolicy

**File**: `k8s/proxynd-networkpolicy.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: proxynd-netpol
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: proxynd
  policyTypes:
    - Ingress
    - Egress

  ingress:
    # Allow from Ingress controller
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080

    # Allow from Prometheus
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
        - podSelector:
            matchLabels:
              app: prometheus
      ports:
        - protocol: TCP
          port: 8080

  egress:
    # Allow to Redis
    - to:
        - podSelector:
            matchLabels:
              app: redis
      ports:
        - protocol: TCP
          port: 6379

    # Allow to Database
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - protocol: TCP
          port: 5432

    # Allow to external package registries
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443  # HTTPS
        - protocol: TCP
          port: 80   # HTTP
```

### 3. Pod Security Standards

Add to namespace:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

---

## 📊 Monitoring Integration

### ServiceMonitor (Prometheus Operator)

**File**: `k8s/proxynd-servicemonitor.yaml`

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: proxynd
  namespace: production
  labels:
    app: proxynd
spec:
  selector:
    matchLabels:
      app: proxynd
  endpoints:
    - port: http
      path: /metrics
      interval: 10s
      scrapeTimeout: 5s
```

---

## 🚀 Deployment Strategies

### 1. Rolling Update (Default)

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0
```

**Pros**:
- Zero downtime
- Gradual rollout
- Easy rollback

**Cons**:
- Both versions run simultaneously
- Slower than recreate

### 2. Blue-Green Deployment

```bash
# Deploy green version
kubectl apply -f k8s/proxynd-deployment-green.yaml

# Switch service to green
kubectl patch service proxynd -p '{"spec":{"selector":{"version":"green"}}}'

# Remove blue version
kubectl delete deployment proxynd-blue
```

### 3. Canary Deployment

```bash
# Deploy canary (10% traffic)
kubectl apply -f k8s/proxynd-deployment-canary.yaml

# Gradually increase canary traffic
# Use Istio or Flagger for sophisticated traffic splitting

# If successful, promote canary to stable
kubectl apply -f k8s/proxynd-deployment-stable.yaml
```

---

## 🔍 Troubleshooting

### Common Issues

**Pods stuck in Pending**:
```bash
kubectl describe pod <pod-name> -n production

# Common causes:
# - Insufficient resources
# - PVC not bound
# - Node selector mismatch
```

**Pods in CrashLoopBackOff**:
```bash
kubectl logs <pod-name> -n production --previous

# Common causes:
# - Config validation failed
# - License missing/invalid
# - Database connection failed
```

**High memory usage**:
```bash
kubectl top pods -n production -l app=proxynd

# Check for memory leaks
# Adjust resource limits if needed
```

---

## 📖 References

- **Kubernetes Best Practices**: https://kubernetes.io/docs/concepts/configuration/overview/
- **Production Checklist**: [production-checklist.md](production-checklist.md)
- **Monitoring**: [../../deployments/prometheus/README.md](../../deployments/prometheus/README.md)
- **Load Testing**: [../../scripts/loadtest/README.md](../../scripts/loadtest/README.md)

---

**Last Updated**: 2025-01-17
**Tested Kubernetes Version**: 1.25-1.29
