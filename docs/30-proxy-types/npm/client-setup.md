# NPM 프록시 설정 가이드

## 클라이언트 설정

### 1. npm 레지스트리 설정

#### 전역 설정
```bash
# 프록시 레지스트리 설정
npm config set registry http://your-proxy-server:8080/proxy/npm/

# HTTPS 강제 비활성화 (HTTP 프록시 사용 시)
npm config set strict-ssl false

# 설정 확인
npm config get registry
```

#### 프로젝트별 설정 (.npmrc)
```ini
# 프로젝트 루트에 .npmrc 파일 생성
registry=http://your-proxy-server:8080/proxy/npm/
strict-ssl=false
```

### 2. yarn 설정

```bash
# yarn 레지스트리 설정
yarn config set registry http://your-proxy-server:8080/proxy/npm/

# 설정 확인
yarn config get registry
```

### 3. pnpm 설정

```bash
# pnpm 레지스트리 설정
pnpm config set registry http://your-proxy-server:8080/proxy/npm/

# 설정 확인
pnpm config get registry
```

## 서버 설정 (ProxyND)

### npm-proxy.yaml 설정 예시

```yaml
path: proxy/npm
use_cache: true

proxies:
  default:
    - name: npmjs
      url: https://registry.npmjs.org
    - name: yarn
      url: https://registry.yarnpkg.com
    - name: taobao
      url: https://registry.npm.taobao.org
```

## 사용 예시

### 패키지 설치
```bash
# 단일 패키지 설치
npm install express

# 여러 패키지 설치
npm install react react-dom

# 개발 의존성 설치
npm install --save-dev webpack webpack-cli

# 전역 패키지 설치
npm install -g typescript
```

### 패키지 검색
```bash
# 패키지 검색
npm search express

# 패키지 정보 보기
npm view express
```

## 스코프 패키지

### 공개 스코프 패키지
```bash
# @types 스코프 패키지
npm install @types/node

# @babel 스코프 패키지
npm install @babel/core @babel/preset-env
```

### 프라이빗 스코프 패키지
```ini
# .npmrc에 프라이빗 레지스트리 설정
@mycompany:registry=http://your-proxy-server:8080/proxy/npm/
//your-proxy-server:8080/proxy/npm/:_authToken=${NPM_TOKEN}
```

## 캐시 동작

- 첫 번째 요청 시 npmjs.org에서 패키지를 다운로드하고 로컬에 캐시
- 이후 요청은 캐시된 파일을 제공
- 패키지 메타데이터는 TTL에 따라 갱신
- tarball URL은 자동으로 프록시 서버로 재작성

## 문제 해결

### SSL 인증서 오류
```bash
# SSL 검증 비활성화 (개발 환경에서만 사용)
npm config set strict-ssl false

# 특정 CA 인증서 설정
npm config set cafile /path/to/ca.pem
```

### 캐시 클리어
```bash
# npm 캐시 클리어
npm cache clean --force

# 서버 캐시 클리어 (ProxyND 서버에서)
rm -rf /storage/proxy/npm/*
```

### 로그 레벨 설정
```bash
# 디버그 로그 활성화
npm install --loglevel=verbose package-name

# 모든 HTTP 요청 로그
npm install --loglevel=http package-name
```

### 프록시 인증
```bash
# 기본 인증 설정 (향후 지원 예정)
npm config set proxy http://username:password@your-proxy-server:8080
npm config set https-proxy http://username:password@your-proxy-server:8080
```

## 성능 최적화

### 병렬 다운로드
```bash
# 동시 연결 수 증가
npm config set maxsockets 10
```

### 타임아웃 설정
```bash
# 타임아웃 증가 (느린 네트워크)
npm config set timeout 60000
```

## CI/CD 환경

### GitHub Actions
```yaml
- name: Setup npm proxy
  run: |
    npm config set registry ${{ secrets.NPM_PROXY_URL }}
    npm config set strict-ssl false
```

### Jenkins
```groovy
sh '''
  npm config set registry http://your-proxy-server:8080/proxy/npm/
  npm config set strict-ssl false
  npm install
'''
```

### Docker
```dockerfile
# Dockerfile
RUN npm config set registry http://your-proxy-server:8080/proxy/npm/ && \
    npm config set strict-ssl false && \
    npm install
```
