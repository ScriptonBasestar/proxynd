# 프론신디 

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
#### env
- CONFIG_DIR=/config
- STORAGE_DIR=/storage
#### 설정파일
`sample-conf` 하위 파일들 참고
- global.yaml
- apt-proxy.yaml
- maven-proxy.yaml

주의 yml 지원안함

### 실행환경

#### docker-compose

```yaml
services:
  deproxy:
    build:
      context: .
      dockerfile: Dockerfile
    image: local_dev/deproxy
    container_name: deproxy
    ports:
      - "8080:8080"
    volumes:
      - ~/tmp/config/:/config
      - storage_volume:/storage
    environment:
      - CONFIG_DIR=/config
      - STORAGE_DIR=/storage

volumes:
  storage_volume:
```

#### helm

helm repo install deproxy https://github.com/ScriptonBasestar-io/deproxy/releases/download

## REF

maven
* https://cwiki.apache.org/confluence/display/MAVENOLD/Repository+Layout+-+Final
* https://cwiki.apache.org/confluence/display/MAVENOLD/Repository+Metadata
apt
* https://wiki.debian.org/DebianRepository/Format
* http://www.ibiblio.org/gferg/ldp/giles/repository/repository-2.html

goproxy
* https://github.com/elazarl/goproxy
