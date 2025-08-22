# Docker 프록시 설정 가이드

## 클라이언트 설정

### 1. Docker 데몬 설정

#### /etc/docker/daemon.json (Linux)
```json
{
  "registry-mirrors": ["http://your-proxy-server:8080/proxy/docker/"],
  "insecure-registries": ["your-proxy-server:8080"]
}
```

#### Docker Desktop 설정 (Windows/Mac)
1. Docker Desktop 설정 열기
2. Docker Engine 탭 선택
3. 위의 JSON 설정 추가
4. Apply & Restart 클릭

### 2. Docker 데몬 재시작

```bash
# Linux
sudo systemctl restart docker

# 또는
sudo service docker restart
```

### 3. 설정 확인

```bash
# Docker 정보 확인
docker info | grep -A 5 "Registry Mirrors"

# 테스트 이미지 pull
docker pull alpine
docker pull nginx:latest
```

## 특정 레지스트리 프록시 사용

### Docker Hub 이미지
```bash
# 프록시를 통한 Docker Hub 이미지 pull
docker pull your-proxy-server:8080/proxy/docker/library/nginx:latest
docker pull your-proxy-server:8080/proxy/docker/library/alpine:3.18
```

### 프라이빗 레지스트리
```bash
# 태그 변경
docker tag myapp:latest your-proxy-server:8080/proxy/docker/mycompany/myapp:latest

# 푸시 (프록시가 쓰기를 지원하는 경우)
docker push your-proxy-server:8080/proxy/docker/mycompany/myapp:latest
```

## 서버 설정 (ProxyND)

### docker-proxy.yaml 설정 예시

```yaml
path: proxy/docker
use_cache: true

proxies:
  - name: docker-hub
    url: https://registry-1.docker.io
  - name: gcr
    url: https://gcr.io
  - name: quay
    url: https://quay.io
  - name: private
    url: https://my-registry.company.com
    auth:
      username: myuser
      password: mypass
```

## Docker Compose 사용

### docker-compose.yml
```yaml
version: '3.8'

services:
  web:
    image: your-proxy-server:8080/proxy/docker/library/nginx:latest
    ports:
      - "80:80"

  db:
    image: your-proxy-server:8080/proxy/docker/library/postgres:15
    environment:
      POSTGRES_PASSWORD: secret
```

## Kubernetes 환경

### containerd 설정
```toml
# /etc/containerd/config.toml
[plugins."io.containerd.grpc.v1.cri".registry.mirrors]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["http://your-proxy-server:8080/proxy/docker/"]
```

### CRI-O 설정
```toml
# /etc/containers/registries.conf
[[registry]]
location = "docker.io"
insecure = true
blocked = false

[[registry.mirror]]
location = "your-proxy-server:8080/proxy/docker"
insecure = true
```

## 인증이 필요한 경우

### Docker 로그인
```bash
# 프록시 서버에 로그인
docker login your-proxy-server:8080/proxy/docker/

# 특정 레지스트리 로그인
docker login your-proxy-server:8080/proxy/docker/gcr.io
```

### 인증 정보 저장
```json
# ~/.docker/config.json
{
  "auths": {
    "your-proxy-server:8080/proxy/docker/": {
      "auth": "base64_encoded_credentials"
    }
  }
}
```

## 캐시 동작

- 이미지 레이어(blob)는 SHA256 해시로 저장되어 중복 제거
- 매니페스트는 태그별로 캐시되며 TTL에 따라 갱신
- 멀티 아키텍처 이미지 지원 (manifest list)

## 문제 해결

### TLS 인증서 오류
```bash
# insecure-registries에 추가
{
  "insecure-registries": ["your-proxy-server:8080"]
}
```

### 캐시 클리어
```bash
# 클라이언트 캐시 클리어
docker system prune -a

# 서버 캐시 클리어 (ProxyND 서버에서)
rm -rf /storage/proxy/docker/*
```

### 디버그 모드
```bash
# Docker 데몬 디버그 모드
dockerd --debug

# 환경 변수로 디버그 활성화
export DOCKER_BUILDKIT=0
docker pull --debug alpine
```

### 프록시 연결 테스트
```bash
# v2 API 엔드포인트 테스트
curl -v http://your-proxy-server:8080/proxy/docker/v2/

# 카탈로그 확인
curl http://your-proxy-server:8080/proxy/docker/v2/_catalog
```

## CI/CD 파이프라인

### GitLab CI
```yaml
variables:
  DOCKER_REGISTRY: your-proxy-server:8080/proxy/docker

before_script:
  - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $DOCKER_REGISTRY

build:
  script:
    - docker build -t $DOCKER_REGISTRY/myapp:$CI_COMMIT_SHA .
    - docker push $DOCKER_REGISTRY/myapp:$CI_COMMIT_SHA
```

### Jenkins
```groovy
pipeline {
  environment {
    DOCKER_REGISTRY = 'your-proxy-server:8080/proxy/docker'
  }

  stages {
    stage('Build') {
      steps {
        script {
          docker.withRegistry("http://${DOCKER_REGISTRY}") {
            def image = docker.build("myapp:${env.BUILD_ID}")
            image.push()
          }
        }
      }
    }
  }
}
```
