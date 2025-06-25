# APT 프록시 설정 가이드

## 클라이언트 설정 (Ubuntu/Debian)

### 1. sources.list 수정

#### Ubuntu 예시
```bash
# /etc/apt/sources.list
deb http://your-proxy-server:8080/proxy/apt/ubuntu focal main restricted universe multiverse
deb http://your-proxy-server:8080/proxy/apt/ubuntu focal-updates main restricted universe multiverse
deb http://your-proxy-server:8080/proxy/apt/ubuntu focal-security main restricted universe multiverse
```

#### Debian 예시
```bash
# /etc/apt/sources.list
deb http://your-proxy-server:8080/proxy/apt/debian bullseye main contrib non-free
deb http://your-proxy-server:8080/proxy/apt/debian bullseye-updates main contrib non-free
deb http://your-proxy-server:8080/proxy/apt/debian bullseye-security main contrib non-free
```

### 2. APT 프록시 환경 변수 설정

```bash
# /etc/apt/apt.conf.d/01proxy
Acquire::http::Proxy "http://your-proxy-server:8080";
Acquire::https::Proxy "http://your-proxy-server:8080";
```

### 3. 사용 예시

```bash
# 패키지 목록 업데이트
sudo apt update

# 패키지 설치
sudo apt install nginx

# 패키지 업그레이드
sudo apt upgrade
```

## 서버 설정 (ProxyND)

### apt-proxy.yaml 설정 예시

```yaml
path: proxy/apt
use_cache: true

proxies:
  ubuntu:
    - name: official
      url: http://archive.ubuntu.com/ubuntu
    - name: kakao
      url: http://mirror.kakao.com/ubuntu
    - name: ubuntu-security
      url: http://security.ubuntu.com/ubuntu
      
  debian:
    - name: official
      url: http://ftp.debian.org/debian
    - name: kakao
      url: http://mirror.kakao.com/debian
```

## 캐시 동작

- 첫 번째 요청 시 upstream 서버에서 파일을 다운로드하고 로컬에 캐시
- 이후 요청은 캐시된 파일을 제공
- Packages, Release 등의 메타데이터 파일은 TTL에 따라 갱신

## 문제 해결

### GPG 키 오류
```bash
# GPG 키 추가
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys [KEY_ID]
```

### 캐시 클리어
```bash
# 클라이언트 캐시 클리어
sudo apt clean

# 서버 캐시 클리어 (ProxyND 서버에서)
rm -rf /storage/proxy/apt/*
```