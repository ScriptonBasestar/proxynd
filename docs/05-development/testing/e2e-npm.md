# NPM Proxy E2E Testing Guide

NPM 프록시를 위한 상세한 End-to-End 테스트 가이드입니다.

## 🎯 테스트 목표

- NPM 레지스트리 프록시 기능 검증
- 패키지 메타데이터 및 tarball 다운로드 확인
- NPM 클라이언트 호환성 검증
- 스코프 패키지 처리 확인

## 📋 전제조건

```bash
# E2E 환경 구성 확인
cd tests/e2e
make status

# NPM 테스트 스크립트 실행 권한 확인
ls -la scripts/test-npm.sh
```

## 🚀 빠른 시작

### 1. NPM E2E 테스트 실행
```bash
cd tests/e2e
make test-npm
```

### 2. 수동 테스트
```bash
# 환경 시작
make up

# NPM 클라이언트 컨테이너 접속
make shell-node

# NPM 설정 확인
npm config get registry

# 테스트 프로젝트에서 패키지 설치
cd /workspace/test-project
npm install express
```

## 📦 테스트 시나리오

### 1. 패키지 메타데이터 조회
```bash
# express 패키지 메타데이터
curl -v "http://proxynd:8080/api/v1/proxy/npm/express"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/json
# - 올바른 package.json 메타데이터
```

### 2. 스코프 패키지 메타데이터
```bash
# @babel/core 스코프 패키지
curl -v "http://proxynd:8080/api/v1/proxy/npm/@babel/core"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/json
# - 스코프 패키지 메타데이터
```

### 3. Tarball 다운로드
```bash
# express tarball 다운로드
curl -v "http://proxynd:8080/api/v1/proxy/npm/express/-/express-4.18.2.tgz"

# 기대 결과:
# - HTTP 200 응답
# - Content-Type: application/octet-stream
# - 유효한 tarball 파일
```

### 4. 스코프 패키지 Tarball
```bash
# @babel/core tarball
curl -v "http://proxynd:8080/api/v1/proxy/npm/@babel/core/-/core-7.22.0.tgz"

# 기대 결과:
# - HTTP 200 응답
# - 올바른 tarball 다운로드
```

### 5. 특정 버전 메타데이터
```bash
# express 특정 버전
curl -v "http://proxynd:8080/api/v1/proxy/npm/express/4.18.2"

# 기대 결과:
# - HTTP 200 응답
# - 해당 버전의 메타데이터
```

### 6. 의존성 해결 테스트
```bash
# package.json으로 전체 의존성 트리 확인
curl -v "http://proxynd:8080/api/v1/proxy/npm/express" | jq '.dependencies'

# 기대 결과:
# - 올바른 의존성 정보
# - 버전 범위 표시
```

### 7. 캐시 동작 확인
```bash
# 첫 번째 요청 (upstream에서 가져옴)
time curl -s "http://proxynd:8080/api/v1/proxy/npm/express" > /dev/null

# 두 번째 요청 (캐시에서 가져옴)
time curl -s "http://proxynd:8080/api/v1/proxy/npm/express" > /dev/null

# 기대 결과:
# - 두 번째 요청이 현저히 빠름
# - 동일한 응답 내용
```

## 🔍 실제 NPM 클라이언트 테스트

### 1. NPM 클라이언트 설정
```bash
# ProxyND를 레지스트리로 설정
npm config set registry http://proxynd:8080/api/v1/proxy/npm

# 설정 확인
npm config get registry
```

### 2. 기본 패키지 설치 테스트
```bash
cd /workspace/test-project

# package.json 생성
npm init -y

# 기본 패키지 설치
npm install express
npm install lodash
npm install moment

# 기대 결과:
# - 성공적인 패키지 설치
# - node_modules 디렉토리 생성
# - package-lock.json 생성
```

### 3. 스코프 패키지 설치
```bash
# Babel 관련 스코프 패키지들
npm install --save-dev @babel/core @babel/cli @babel/preset-env

# 기대 결과:
# - 스코프 패키지 정상 설치
# - 올바른 디렉토리 구조 생성
```

### 4. 개발 의존성 설치
```bash
# 개발 의존성 설치
npm install --save-dev jest eslint

# 기대 결과:
# - devDependencies에 추가
# - 정상 설치 완료
```

### 5. 전역 패키지 설치
```bash
# 전역 패키지 설치 (선택사항)
npm install -g nodemon

# 기대 결과:
# - 전역 패키지 설치 성공
```

### 6. 패키지 정보 조회
```bash
# 패키지 정보 확인
npm info express
npm info @babel/core

# 기대 결과:
# - 상세한 패키지 정보 표시
# - 버전, 의존성, 설명 등
```

## 📊 성능 테스트

### 1. 다중 패키지 동시 설치
```bash
# 여러 패키지 동시 설치
npm install express lodash axios moment chalk

# 기대 결과:
# - 모든 패키지 성공적 설치
# - 합리적인 설치 시간
```

### 2. 대규모 프로젝트 의존성
```bash
# React 프로젝트 의존성 (많은 의존성을 가진 예시)
npm install react react-dom

# 기대 결과:
# - 안정적인 의존성 해결
# - 타임아웃 없는 설치
```

### 3. 캐시 성능 비교
```bash
# 캐시 클리어 후 설치
npm cache clean --force
rm -rf node_modules package-lock.json
time npm install

# 캐시된 상태에서 재설치
rm -rf node_modules
time npm install

# 기대 결과:
# - 두 번째 설치가 현저히 빠름
```

## 🔧 문제 해결

### 1. 패키지를 찾을 수 없음 (404)
```bash
# ProxyND 로그 확인
make logs-proxynd | grep npm

# upstream 연결 확인
curl -v http://nginx-upstream:8081/npm/express

# 해결책:
# - upstream 데이터 확인
# - 네트워크 연결 확인
# - 프록시 설정 검토
```

### 2. Tarball 다운로드 실패
```bash
# tarball 직접 접근 테스트
curl -I "http://proxynd:8080/api/v1/proxy/npm/express/-/express-4.18.2.tgz"

# 해결책:
# - URL 경로 확인
# - 파일 존재 여부 확인
# - 권한 설정 검토
```

### 3. 스코프 패키지 오류
```bash
# 스코프 패키지 URL 인코딩 확인
curl -v "http://proxynd:8080/api/v1/proxy/npm/%40babel%2Fcore"

# 해결책:
# - URL 인코딩 처리 확인
# - 스코프 패키지 설정 검토
```

### 4. NPM 클라이언트 연결 오류
```bash
# NPM 로그 활성화
npm install express --loglevel=verbose

# 해결책:
# - 레지스트리 URL 확인
# - 네트워크 연결 테스트
# - 프록시 설정 검증
```

## 📈 모니터링

### 1. 메트릭 확인 (구현된 경우)
```bash
# NPM 프록시 메트릭
curl http://proxynd:8080/metrics | grep npm

# 기대 메트릭:
# - npm_requests_total
# - npm_cache_hits_total
# - npm_package_downloads_total
```

### 2. 로그 분석
```bash
# NPM 관련 로그 필터링
make logs-proxynd | grep -i npm

# 패키지 다운로드 패턴 분석
grep "GET.*npm" /storage/access.log | head -10
```

### 3. 패키지 다운로드 통계
```bash
# 인기 패키지 확인
grep "npm.*tarball" /storage/access.log | awk '{print $7}' | sort | uniq -c | sort -nr | head -10
```

## ✅ 성공 기준

- [ ] 패키지 메타데이터 올바른 제공
- [ ] Tarball 다운로드 성공
- [ ] 스코프 패키지 완전 지원
- [ ] NPM 클라이언트 완전 호환성
- [ ] 의존성 해결 정상 동작
- [ ] 캐시 기능 정상 동작
- [ ] 동시 설치 안정적 처리
- [ ] 대규모 프로젝트 지원

## 🧪 추가 테스트 케이스

### 1. 버전 범위 테스트
```json
{
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "~4.17.0",
    "moment": ">=2.29.0 <3.0.0"
  }
}
```

### 2. Peer Dependencies 테스트
```bash
# peer dependency 경고 확인
npm install react-router-dom
```

### 3. 옵셔널 Dependencies 테스트
```bash
# optional dependency 처리
npm install fsevents --optional
```

## 🔄 자동화

이 테스트들은 `tests/e2e/scripts/test-npm.sh`에 자동화되어 있으며, CI/CD 파이프라인에서 자동 실행됩니다.

```bash
# 전체 자동화 테스트 실행
cd tests/e2e
./scripts/test-npm.sh

# 기대 출력:
# ✅ NPM proxy basic functionality
# ✅ NPM client compatibility
# ✅ Scoped packages support
# ✅ Cache behavior
# ✅ Performance requirements
```
