# ProxyND Helm Chart

ProxyND를 Kubernetes에 배포하기 위한 Helm 차트입니다.

## 전제 조건

- Kubernetes 1.20+
- Helm 3.0+
- PV provisioner (영구 저장소 사용 시)

## 설치

### 차트 저장소 추가

```bash
helm repo add proxynd https://scriptonbasestar.github.io/proxynd
helm repo update
```

### 기본 설치

```bash
helm install proxynd proxynd/proxynd
```

### 사용자 정의 값으로 설치

```bash
# values 파일 사용
helm install proxynd proxynd/proxynd -f my-values.yaml

# 명령줄 옵션 사용
helm install proxynd proxynd/proxynd \
  --set image.tag=v1.0.0 \
  --set persistence.size=50Gi
```

### 특정 네임스페이스에 설치

```bash
kubectl create namespace proxynd
helm install proxynd proxynd/proxynd -n proxynd
```

## 설정

### 기본 설정

```yaml
# 이미지 설정
image:
  repository: scriptonbasestar/proxynd
  tag: latest
  pullPolicy: IfNotPresent

# ProxyND 설정
proxynd:
  serverPort: 8080
  logLevel: info
  logFormat: json
  
  # 캐시 설정
  cache:
    type: filesystem  # filesystem 또는 s3
    ttl: 3600
    maxSize: "10Gi"
```

### 프록시 설정

각 패키지 매니저별 프록시 활성화:

```yaml
proxynd:
  proxies:
    npm:
      enabled: true
      upstream: https://registry.npmjs.org
    
    pip:
      enabled: true
      upstream: https://pypi.org/simple
    
    apt:
      enabled: true
      distributions:
        - name: ubuntu
          upstream: http://archive.ubuntu.com/ubuntu
          architectures: ["amd64", "arm64"]
    
    docker:
      enabled: true
      upstream: https://registry-1.docker.io
```

### 인증 설정

Basic 인증 활성화:

```yaml
proxynd:
  auth:
    enabled: true
    type: basic
    basicAuth:
      users:
        - username: admin
          password: secretpassword
          permissions: ["read", "write", "delete"]
        - username: readonly
          password: readonlypass
          permissions: ["read"]
```

### 영구 저장소

```yaml
persistence:
  enabled: true
  storageClass: "fast-ssd"
  accessModes:
    - ReadWriteOnce
  size: 100Gi
  
  # 기존 PVC 사용
  # existingClaim: my-existing-pvc
```

### S3 캐시 백엔드

```yaml
proxynd:
  cache:
    type: s3
    s3:
      bucket: proxynd-cache
      region: ap-northeast-2
      # S3 호환 스토리지용 엔드포인트
      # endpoint: https://minio.example.com
```

### Ingress 설정

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: "0"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
  hosts:
    - host: proxynd.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: proxynd-tls
      hosts:
        - proxynd.example.com
```

### 리소스 제한

```yaml
resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 500m
    memory: 1Gi
```

### 고가용성 설정

```yaml
# 복제본 수
replicaCount: 3

# Pod 분산 배치
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - proxynd
        topologyKey: kubernetes.io/hostname

# 토폴로지 분산
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: proxynd
```

### 모니터링

Prometheus ServiceMonitor 활성화:

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      prometheus: kube-prometheus
```

### 보안 설정

```yaml
# Pod 보안 컨텍스트
podSecurityContext:
  enabled: true
  fsGroup: 65534
  runAsUser: 65534
  runAsNonRoot: true

# 컨테이너 보안 컨텍스트
containerSecurityContext:
  enabled: true
  runAsUser: 65534
  runAsNonRoot: true
  readOnlyRootFilesystem: false
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL

# 네트워크 정책
networkPolicy:
  enabled: true
  allowExternal: true
```

## 업그레이드

```bash
# 차트 업데이트
helm repo update

# 업그레이드
helm upgrade proxynd proxynd/proxynd

# 특정 버전으로 업그레이드
helm upgrade proxynd proxynd/proxynd --version 0.2.0
```

## 롤백

```bash
# 이전 릴리스로 롤백
helm rollback proxynd

# 특정 리비전으로 롤백
helm rollback proxynd 3
```

## 제거

```bash
helm uninstall proxynd

# 네임스페이스도 함께 제거
helm uninstall proxynd -n proxynd
kubectl delete namespace proxynd
```

## 문제 해결

### 차트 값 확인

```bash
# 기본 값 확인
helm show values proxynd/proxynd

# 설치된 릴리스 값 확인
helm get values proxynd
```

### Pod 상태 확인

```bash
kubectl get pods -l app.kubernetes.io/name=proxynd
kubectl describe pod proxynd-xxxxx
kubectl logs proxynd-xxxxx
```

### 이벤트 확인

```bash
kubectl get events --sort-by='.lastTimestamp'
```

## 예제 설정

### 개발 환경

```yaml
# values-dev.yaml
replicaCount: 1

image:
  pullPolicy: Always

proxynd:
  logLevel: debug
  
persistence:
  enabled: false

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

### 프로덕션 환경

```yaml
# values-prod.yaml
replicaCount: 3

image:
  tag: v1.0.0
  pullPolicy: IfNotPresent

proxynd:
  logLevel: info
  cache:
    type: s3
    maxSize: "100Gi"

persistence:
  enabled: true
  storageClass: fast-ssd
  size: 200Gi

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: registry.company.com

resources:
  limits:
    cpu: 4000m
    memory: 8Gi
  requests:
    cpu: 1000m
    memory: 2Gi

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app.kubernetes.io/name: proxynd
        topologyKey: kubernetes.io/hostname
```

## 차트 개발

### 로컬 테스트

```bash
# 템플릿 렌더링
helm template proxynd ./helm

# 문법 검사
helm lint ./helm

# Dry-run
helm install proxynd ./helm --dry-run --debug
```

### 차트 패키징

```bash
helm package ./helm
```

### 차트 버전 관리

Chart.yaml에서 버전 업데이트:

```yaml
version: 0.1.4  # 차트 버전
appVersion: "0.1.4"  # 애플리케이션 버전
```