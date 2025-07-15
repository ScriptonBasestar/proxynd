# ProxyND 배포 가이드

ProxyND를 다양한 환경에 배포하는 방법을 설명합니다.

## 목차

1. [Docker 배포](#docker-배포)
2. [Kubernetes 배포 (Helm)](#kubernetes-배포-helm)
3. [Systemd 서비스 배포](#systemd-서비스-배포)
4. [Docker Compose 배포](#docker-compose-배포)
5. [클라우드 플랫폼별 배포](#클라우드-플랫폼별-배포)

## Docker 배포

### 단일 컨테이너 실행

```bash
# 기본 실행
docker run -d \
  --name proxynd \
  -p 8080:8080 \
  -v ./config:/config \
  -v proxynd-storage:/storage \
  scriptonbasestar/proxynd:latest

# 환경 변수 설정
docker run -d \
  --name proxynd \
  -p 8080:8080 \
  -v ./config:/config \
  -v proxynd-storage:/storage \
  -e LOG_LEVEL=debug \
  -e CACHE_TYPE=s3 \
  -e S3_BUCKET=proxynd-cache \
  scriptonbasestar/proxynd:latest
```

### Docker 네트워크 설정

```bash
# 커스텀 네트워크 생성
docker network create proxynd-net

# 네트워크와 함께 실행
docker run -d \
  --name proxynd \
  --network proxynd-net \
  -p 8080:8080 \
  -v ./config:/config \
  -v proxynd-storage:/storage \
  scriptonbasestar/proxynd:latest
```

## Kubernetes 배포 (Helm)

### Helm 차트 설치

```bash
# 차트 저장소 추가
helm repo add proxynd https://scriptonbasestar.github.io/proxynd
helm repo update

# 기본 설치
helm install proxynd proxynd/proxynd

# 커스텀 values로 설치
helm install proxynd proxynd/proxynd \
  --namespace proxynd \
  --create-namespace \
  -f values-prod.yaml
```

### 수동 Kubernetes 배포

```yaml
# proxynd-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxynd
  namespace: default
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
        image: scriptonbasestar/proxynd:latest
        ports:
        - containerPort: 8080
        env:
        - name: SERVER_PORT
          value: "8080"
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
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: proxynd-config
      - name: storage
        persistentVolumeClaim:
          claimName: proxynd-storage
---
apiVersion: v1
kind: Service
metadata:
  name: proxynd
spec:
  selector:
    app: proxynd
  ports:
  - port: 8080
    targetPort: 8080
  type: LoadBalancer
```

## Systemd 서비스 배포

### 자동 설치

```bash
# 바이너리 빌드
go build -o proxynd main.go

# 서비스 설치
sudo ./systemd/install.sh -b ./proxynd -c ./sample-conf

# 서비스 시작
sudo systemctl start proxynd
sudo systemctl enable proxynd
```

### 다중 인스턴스 배포

```bash
# NPM 전용 인스턴스
sudo ./systemd/install.sh -b ./proxynd -i npm -p 8081

# Docker 전용 인스턴스  
sudo ./systemd/install.sh -b ./proxynd -i docker -p 8082

# 인스턴스 관리
sudo systemctl status proxynd@npm
sudo systemctl status proxynd@docker
```

## Docker Compose 배포

### 기본 구성

```yaml
# docker-compose.yml
version: '3.8'

services:
  proxynd:
    image: scriptonbasestar/proxynd:latest
    container_name: proxynd
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    volumes:
      - ./config:/config:ro
      - proxynd-storage:/storage
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped

volumes:
  proxynd-storage:
    driver: local
```

### 프로덕션 구성

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  proxynd:
    image: scriptonbasestar/proxynd:v1.0.0
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    ports:
      - target: 8080
        published: 8080
        protocol: tcp
        mode: host
    environment:
      - SERVER_PORT=8080
      - LOG_LEVEL=info
      - CACHE_TYPE=s3
      - S3_BUCKET=proxynd-cache
      - S3_REGION=ap-northeast-2
    volumes:
      - ./config:/config:ro
      - proxynd-storage:/storage
    networks:
      - proxynd-net
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - proxynd
    networks:
      - proxynd-net

networks:
  proxynd-net:
    driver: overlay

volumes:
  proxynd-storage:
    driver: local
```

## 클라우드 플랫폼별 배포

### AWS ECS

```json
{
  "family": "proxynd",
  "taskDefinition": {
    "containerDefinitions": [
      {
        "name": "proxynd",
        "image": "scriptonbasestar/proxynd:latest",
        "portMappings": [
          {
            "containerPort": 8080,
            "protocol": "tcp"
          }
        ],
        "environment": [
          {
            "name": "SERVER_PORT",
            "value": "8080"
          },
          {
            "name": "CACHE_TYPE",
            "value": "s3"
          }
        ],
        "mountPoints": [
          {
            "sourceVolume": "config",
            "containerPath": "/config"
          }
        ],
        "healthCheck": {
          "command": ["CMD-SHELL", "wget -q --spider http://localhost:8080/healthz || exit 1"],
          "interval": 30,
          "timeout": 5,
          "retries": 3
        }
      }
    ],
    "volumes": [
      {
        "name": "config",
        "host": {
          "sourcePath": "/ecs/config"
        }
      }
    ]
  }
}
```

### Google Cloud Run

```bash
# 이미지 빌드 및 푸시
gcloud builds submit --tag gcr.io/PROJECT-ID/proxynd

# Cloud Run 배포
gcloud run deploy proxynd \
  --image gcr.io/PROJECT-ID/proxynd \
  --platform managed \
  --region asia-northeast3 \
  --port 8080 \
  --memory 2Gi \
  --cpu 2 \
  --max-instances 10 \
  --set-env-vars="LOG_LEVEL=info,CACHE_TYPE=gcs"
```

### Azure Container Instances

```bash
# 리소스 그룹 생성
az group create --name proxynd-rg --location koreacentral

# 컨테이너 인스턴스 생성
az container create \
  --resource-group proxynd-rg \
  --name proxynd \
  --image scriptonbasestar/proxynd:latest \
  --cpu 2 \
  --memory 4 \
  --ports 8080 \
  --environment-variables \
    SERVER_PORT=8080 \
    LOG_LEVEL=info \
    CACHE_TYPE=azure
```

## 리버스 프록시 설정

### Nginx

```nginx
upstream proxynd {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081 backup;
}

server {
    listen 80;
    server_name registry.example.com;

    location / {
        proxy_pass http://proxynd;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 대용량 파일 지원
        client_max_body_size 0;
        proxy_buffering off;
        proxy_request_buffering off;
    }
    
    location /healthz {
        proxy_pass http://proxynd/healthz;
        access_log off;
    }
}
```

### HAProxy

```haproxy
global
    maxconn 4096
    log stdout local0

defaults
    mode http
    timeout connect 5s
    timeout client 30s
    timeout server 30s
    option httplog

frontend proxynd_frontend
    bind *:80
    default_backend proxynd_backend

backend proxynd_backend
    balance roundrobin
    option httpchk GET /healthz
    server proxynd1 127.0.0.1:8080 check
    server proxynd2 127.0.0.1:8081 check backup
```

## 모니터링 설정

### Prometheus 연동

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'proxynd'
    static_configs:
      - targets: ['proxynd:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Grafana 대시보드

ProxyND 메트릭을 위한 Grafana 대시보드 ID: `14567`

```bash
# Grafana 대시보드 임포트
curl -X POST http://admin:admin@localhost:3000/api/dashboards/import \
  -H "Content-Type: application/json" \
  -d '{"dashboard": {"id": 14567}, "overwrite": true}'
```

## 보안 고려사항

### TLS 설정

```yaml
# Kubernetes Ingress TLS
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: proxynd-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - registry.example.com
    secretName: proxynd-tls
  rules:
  - host: registry.example.com
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

### 방화벽 규칙

```bash
# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -j DROP

# ufw
sudo ufw allow from 10.0.0.0/8 to any port 8080
sudo ufw deny 8080
```

## 백업 및 복구

### 캐시 백업

```bash
# 로컬 파일시스템 캐시 백업
tar -czf proxynd-cache-$(date +%Y%m%d).tar.gz /var/lib/proxynd

# S3 캐시 동기화
aws s3 sync /var/lib/proxynd s3://backup-bucket/proxynd/
```

### 설정 백업

```bash
# 설정 백업
tar -czf proxynd-config-$(date +%Y%m%d).tar.gz /etc/proxynd

# Git 관리
cd /etc/proxynd
git init
git add .
git commit -m "Configuration backup $(date)"
git remote add origin https://github.com/company/proxynd-config.git
git push origin main
```

## 문제 해결

### 성능 튜닝

```bash
# 시스템 리소스 확인
docker stats proxynd

# 파일 디스크립터 제한 증가
ulimit -n 65536

# 커널 파라미터 조정
echo "net.core.somaxconn = 65535" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65535" >> /etc/sysctl.conf
sysctl -p
```

### 디버깅

```bash
# 상세 로그 활성화
export LOG_LEVEL=debug

# 실시간 로그 확인
docker logs -f proxynd

# 메트릭 확인
curl http://localhost:8080/metrics | grep proxynd

# 헬스체크
curl -v http://localhost:8080/healthz
```