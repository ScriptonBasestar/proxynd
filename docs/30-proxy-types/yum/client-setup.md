# YUM/DNF 클라이언트 설정 가이드

이 문서는 YUM/DNF 패키지 매니저가 ProxyND를 통해 패키지를 다운로드하도록 설정하는 방법을 설명합니다.

## 지원 OS

- CentOS 7/8
- RHEL 7/8/9
- Rocky Linux 8/9
- AlmaLinux 8/9
- Oracle Linux 7/8/9

## 설정 방법

### 1. YUM 저장소 설정 백업

```bash
# 기존 repo 파일 백업
sudo cp -r /etc/yum.repos.d /etc/yum.repos.d.backup
```

### 2. ProxyND를 사용하도록 저장소 설정 수정

#### 방법 1: sed를 사용한 일괄 변경

```bash
# ProxyND 서버 주소 설정
PROXYND_URL="http://your-proxynd-server:8080/proxy/yum"

# 모든 repo 파일의 baseurl을 ProxyND로 변경
sudo sed -i.bak "s|^baseurl=http://|baseurl=${PROXYND_URL}/|g" /etc/yum.repos.d/*.repo
sudo sed -i "s|^baseurl=https://|baseurl=${PROXYND_URL}/|g" /etc/yum.repos.d/*.repo
```

#### 방법 2: 개별 저장소 파일 수정

`/etc/yum.repos.d/CentOS-Base.repo` 예시:

```ini
[base]
name=CentOS-$releasever - Base
# 원본 URL을 주석 처리
#mirrorlist=http://mirrorlist.centos.org/?release=$releasever&arch=$basearch&repo=os&infra=$infra
#baseurl=http://mirror.centos.org/centos/$releasever/os/$basearch/

# ProxyND URL로 변경
baseurl=http://your-proxynd-server:8080/proxy/yum/centos/$releasever/os/$basearch/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7
```

### 3. YUM 캐시 정리 및 테스트

```bash
# 캐시 정리
sudo yum clean all

# 메타데이터 갱신
sudo yum makecache

# 패키지 설치 테스트
sudo yum install -y tree
```

## 고급 설정

### HTTP 프록시 설정 (선택사항)

`/etc/yum.conf`에 프록시 설정 추가:

```ini
[main]
cachedir=/var/cache/yum/$basearch/$releasever
keepcache=0
debuglevel=2
logfile=/var/log/yum.log
exactarch=1
obsoletes=1
gpgcheck=1
plugins=1
installonly_limit=5

# ProxyND가 프록시 역할을 하는 경우
proxy=http://your-proxynd-server:8080
```

### 특정 저장소만 ProxyND 사용

특정 저장소만 ProxyND를 사용하려면 해당 repo 파일만 수정:

```ini
[epel]
name=Extra Packages for Enterprise Linux 7 - $basearch
baseurl=http://your-proxynd-server:8080/proxy/yum/epel/7/$basearch
enabled=1
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-7
```

## 문제 해결

### 1. GPG 키 오류

```bash
# GPG 키 다시 가져오기
sudo rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-*
```

### 2. 연결 오류

```bash
# ProxyND 서버 연결 테스트
curl -I http://your-proxynd-server:8080/healthz

# 저장소 메타데이터 직접 테스트
curl http://your-proxynd-server:8080/proxy/yum/centos/7/os/x86_64/repodata/repomd.xml
```

### 3. 디버그 모드 실행

```bash
# 상세 로그 출력
sudo yum -v update
```

## 원래 설정으로 복구

```bash
# 백업한 설정으로 복구
sudo rm -rf /etc/yum.repos.d
sudo mv /etc/yum.repos.d.backup /etc/yum.repos.d
sudo yum clean all
```

## DNF (Fedora/RHEL 8+) 설정

DNF는 YUM의 후속 버전으로 동일한 설정 방법을 사용합니다:

```bash
# DNF 설정 파일 위치
/etc/dnf/dnf.conf
/etc/yum.repos.d/*.repo  # YUM과 동일

# 캐시 정리 및 테스트
sudo dnf clean all
sudo dnf makecache
sudo dnf install -y tree
```
