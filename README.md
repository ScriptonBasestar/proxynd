# 프론신디 

## 지원기능

- [x] Maven Proxy
- [ ] Maven Mirror
- [x] Apt Proxy
- [ ] Apt Mirror

- [ ] Npm
- [ ] Go
- [ ] Python
- [ ] Ruby
- [ ] Docker
- [ ] 쩌 Repo 인증기능
- [ ] 이 리포 인증기능 

## Install

### 설정

기본값 없음을 지향함
기본값이 편한듯 하기도 하지만 이 때문에 오류가 발생하는 경우가 많고 이런경우 문제해결이 매우 힘듦

#### env
- CONFIG_DIR=/config
- STORAGE_DIR=/storage
- SERVER_PORT=8080

#### 설정파일
`sample-conf` 하위 파일들 참고
- global.yaml
- apt-proxy.yaml
- maven-proxy.yaml

**주의! .yml 지원안함**

### 실행

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
