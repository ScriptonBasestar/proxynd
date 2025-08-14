# APT Proxy E2E Testing Guide

APT 프록시를 위한 상세한 End-to-End 테스트 가이드입니다.

## 🎯 테스트 목표

- APT 패키지 저장소 프록시 기능 검증
- Release/Packages 파일 처리 확인
- .deb 패키지 다운로드 및 캐시 동작 확인
- APT 클라이언트 호환성 검증

## 📋 전제조건

```bash
# E2E 환경 구성 확인
cd tests/e2e
make status

# APT 테스트 스크립트 실행 권한 확인
ls -la scripts/test-apt.sh
```

## 🚀 빠른 시작

### 1. APT E2E 테스트 실행
```bash
cd tests/e2e
make test-apt
```

### 2. 수동 테스트
```bash
# 환경 시작
make up

# Ubuntu 클라이언트 컨테이너 접속
make shell-ubuntu

# APT 소스 리스트 확인
cat /etc/apt/sources.list.d/proxynd.list

# 패키지 목록 업데이트 테스트
apt update
```

## 📦 테스트 시나리오

### 1. Release 파일 조회
```bash
# Ubuntu jammy Release 파일
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/Release"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: text/plain
# - 올바른 Release 파일 내용
```

### 2. Release.gpg 서명 파일
```bash
# GPG 서명 파일
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/Release.gpg"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/octet-stream
# - 바이너리 GPG 서명 데이터
```

### 3. InRelease 파일 (서명된 Release)
```bash
# InRelease 파일 (서명이 포함된 Release)
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/InRelease"

# 기대 결과:
# - HTTP 200 응답
# - PGP 서명이 포함된 Release 내용
```

### 4. Packages 파일
```bash
# main 컴포넌트 Packages 파일
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages"

# 압축된 Packages.gz 파일
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages.gz"

# 기대 결과:
# - HTTP 200 응답
# - 올바른 패키지 목록 데이터
```

### 5. .deb 패키지 다운로드
```bash
# 실제 .deb 파일 다운로드 (curl 패키지 예시)
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/pool/main/c/curl/curl_7.81.0-1ubuntu1.15_amd64.deb"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/vnd.debian.binary-package
# - 유효한 .deb 파일
```

### 6. 다양한 아키텍처 지원
```bash
# arm64 아키텍처 Packages 파일
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/main/binary-arm64/Packages"

# i386 아키텍처
curl -v "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/main/binary-i386/Packages"

# 기대 결과:
# - HTTP 200 응답
# - 아키텍처별 패키지 목록
```

### 7. 캐시 동작 확인
```bash
# 첫 번째 요청 (upstream에서 가져옴)
time curl -s "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/Release" > /dev/null

# 두 번째 요청 (캐시에서 가져옴)
time curl -s "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/Release" > /dev/null

# 기대 결과:
# - 두 번째 요청이 현저히 빠름
# - 동일한 응답 내용
```

## 🔍 실제 APT 클라이언트 테스트

### 1. APT 소스 설정
```bash
# ProxyND를 APT 소스로 설정
echo "deb http://proxynd:8080/api/v1/proxy/apt/ubuntu jammy main" > /etc/apt/sources.list.d/proxynd.list

# 또는 기본 소스 교체
cat > /etc/apt/sources.list << EOF
deb http://proxynd:8080/api/v1/proxy/apt/ubuntu jammy main restricted
deb http://proxynd:8080/api/v1/proxy/apt/ubuntu jammy universe multiverse
deb http://proxynd:8080/api/v1/proxy/apt/ubuntu jammy-updates main restricted
EOF
```

### 2. 패키지 목록 업데이트
```bash
# 패키지 목록 업데이트
apt update

# 기대 결과:
# - 성공적인 패키지 목록 다운로드
# - GPG 키 검증 성공 (설정된 경우)
# - "Reading package lists... Done" 메시지
```

### 3. 패키지 검색
```bash
# 패키지 검색
apt search curl
apt show curl

# 기대 결과:
# - 정확한 패키지 정보 표시
# - 의존성 정보 포함
```

### 4. 패키지 설치
```bash
# 기본 패키지 설치
apt install -y curl wget nano

# 기대 결과:
# - 성공적인 패키지 다운로드 및 설치
# - 의존성 자동 해결
```

### 5. 패키지 업그레이드
```bash
# 시스템 업그레이드 확인
apt list --upgradable
apt upgrade -y

# 기대 결과:
# - 업그레이드 가능한 패키지 목록
# - 성공적인 업그레이드 처리
```

### 6. 캐시 정보 확인
```bash
# APT 캐시 통계
apt-cache stats
apt-cache policy

# 기대 결과:
# - 올바른 저장소 정보
# - 우선순위 설정 확인
```

## 📊 성능 테스트

### 1. 대규모 패키지 목록 다운로드
```bash
# 전체 universe 컴포넌트 (큰 Packages 파일)
time curl -s "http://proxynd:8080/api/v1/proxy/apt/ubuntu/dists/jammy/universe/binary-amd64/Packages.gz" > /dev/null

# 기대 결과:
# - 안정적인 대용량 파일 다운로드
# - 합리적인 다운로드 시간
```

### 2. 동시 apt 업데이트
```bash
# 여러 터미널에서 동시 apt update 실행
seq 1 5 | xargs -n1 -P5 bash -c 'apt update > /dev/null 2>&1'

# 기대 결과:
# - 모든 업데이트 성공
# - 동시성 문제 없음
```

### 3. 캐시 성능 측정
```bash
# 캐시 미스 시간 측정
time apt update

# 캐시 히트 시간 측정 (즉시 재실행)
time apt update

# 기대 결과:
# - 두 번째 실행이 현저히 빠름
```

## 🔧 문제 해결

### 1. Release 파일을 찾을 수 없음
```bash
# ProxyND 로그 확인
make logs-proxynd | grep apt

# upstream 연결 확인
curl -v http://nginx-upstream:8081/apt/ubuntu/dists/jammy/Release

# 해결책:
# - upstream 데이터 확인
# - 디스트리뷰션 이름 확인
# - 경로 매핑 검토
```

### 2. GPG 검증 실패
```bash
# GPG 키 확인
gpg --list-keys

# APT 키 추가 (필요한 경우)
apt-key add /etc/apt/trusted.gpg.d/ubuntu-archive-keyring.gpg

# 또는 검증 비활성화 (테스트 목적)
echo 'APT::Get::AllowUnauthenticated "true";' > /etc/apt/apt.conf.d/99-allow-unauthenticated

# 해결책:
# - 올바른 GPG 키 설정
# - 서명 파일 확인
# - 키 관리 프로세스 검토
```

### 3. .deb 다운로드 실패
```bash
# 패키지 경로 확인
apt-cache policy curl
apt-get download --print-uris curl

# 해결책:
# - 패키지 풀 경로 확인
# - 파일명 및 버전 검증
# - 스토리지 권한 확인
```

### 4. 아키텍처 관련 오류
```bash
# 지원 아키텍처 확인
dpkg --print-architecture
dpkg --print-foreign-architectures

# 해결책:
# - 아키텍처별 패키지 확인
# - 멀티아키텍처 설정 검토
```

## 📈 모니터링

### 1. 메트릭 확인 (구현된 경우)
```bash
# APT 프록시 메트릭
curl http://proxynd:8080/metrics | grep apt

# 기대 메트릭:
# - apt_requests_total
# - apt_cache_hits_total
# - apt_package_downloads_total
# - apt_release_file_updates
```

### 2. 로그 분석
```bash
# APT 관련 로그 필터링
make logs-proxynd | grep -i apt

# Release 파일 액세스 패턴
grep "apt.*Release" /storage/access.log | head -10

# .deb 다운로드 통계
grep "apt.*\.deb" /storage/access.log | wc -l
```

### 3. 저장소 사용 통계
```bash
# 인기 패키지 확인
grep "apt.*pool.*\.deb" /storage/access.log | awk -F'/' '{print $(NF)}' | sort | uniq -c | sort -nr | head -10
```

## ✅ 성공 기준

- [ ] Release 파일 올바른 제공
- [ ] GPG 서명 파일 정상 처리
- [ ] Packages 파일 완전 지원
- [ ] .deb 패키지 다운로드 성공
- [ ] APT 클라이언트 완전 호환성
- [ ] 멀티 아키텍처 지원
- [ ] 캐시 기능 정상 동작
- [ ] 동시 접근 안정적 처리

## 🧪 추가 테스트 케이스

### 1. PPA 저장소 테스트
```bash
# 예: Deadsnakes PPA
echo "deb http://proxynd:8080/api/v1/proxy/apt/ppa/deadsnakes/ppa jammy main" >> /etc/apt/sources.list.d/deadsnakes.list
```

### 2. 보안 업데이트 저장소
```bash
# Ubuntu 보안 업데이트
echo "deb http://proxynd:8080/api/v1/proxy/apt/ubuntu jammy-security main" >> /etc/apt/sources.list.d/security.list
```

### 3. 소스 패키지 지원
```bash
# 소스 패키지 저장소
echo "deb-src http://proxynd:8080/api/v1/proxy/apt/ubuntu jammy main" >> /etc/apt/sources.list.d/source.list
```

## 🔄 자동화

이 테스트들은 `tests/e2e/scripts/test-apt.sh`에 자동화되어 있으며, CI/CD 파이프라인에서 자동 실행됩니다.

```bash
# 전체 자동화 테스트 실행
cd tests/e2e
./scripts/test-apt.sh

# 기대 출력:
# ✅ APT proxy basic functionality
# ✅ APT client compatibility
# ✅ Package management operations
# ✅ GPG verification support
# ✅ Cache behavior
# ✅ Performance requirements
```
