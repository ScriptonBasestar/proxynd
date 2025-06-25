# 프록신디 

## 지원기능

- [x] Maven Proxy
- [x] Apt Proxy
- [ ] Npm
- [ ] Python
- [ ] Ruby
- [ ] 쩌 Repo 인증기능
- [ ] 이 리포 인증기능 
 
불가능
- Docker - distribution/registry를 연동하는 방법
- Golang - https://github.com/gomods/athens.git

## Install

### 설정

설정파일이 좀 복잡하면 어때서
반복적이고 모든 내용이 다 적혀있는 설정이 최고다

#### 설정파일
default.yaml > server1.yaml 와같은 형태로 오버라이딩

yaml, yml, toml 순서로 로딩

**`sample-conf` 디렉토리 하위 파일들 참고**

- global.yaml
- apt-proxy.yaml
- maven-proxy.yaml

#### env 오버라이딩
- CONFIG_DIR=/config
- STORAGE_DIR=/storage
- SERVER_PORT=8080

### 실행

#### main run 

```bash
set -a; source .env; set +a

main run main.go --config-dir /config --storage-dir /storage --server-port 8080
```

#### docker-compose

```yaml
services:
  proxynd:
    build:
      context: .
      dockerfile: Dockerfile
    image: local_dev/proxynd
    container_name: proxynd
    ports:
      - "8080:8080"
    volumes:
      - ~/tmp/config/:/config
      - storage_volume:/storage
    environment:
      - CONFIG_DIR=/config
      - STORAGE_DIR=/storage
      - SERVER_HOST=localhost
      - SERVER_PORT=8080 

volumes:
  storage_volume:
```

#### helm

안될수도 있음
kube 1.20 이상

```bash
helm repo install proxynd https://github.com/ScriptonBasestar-io/proxynd/releases/download
