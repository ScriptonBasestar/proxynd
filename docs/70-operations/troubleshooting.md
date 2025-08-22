# 🔧 ProxyND 운영 문제 해결

ProxyND 운영 중 발생할 수 있는 일반적인 문제들과 해결 방법을 안내합니다. 모니터링에서 감지된 문제들의 진단과 해결에 집중합니다.

## 🚨 일반적인 문제들

### 1. 서버 시작 오류

#### CONFIG_DIR 또는 STORAGE_DIR 미설정
```bash
# 증상
FATAL error: CONFIG_DIR environment variable is required

# 해결책
export CONFIG_DIR=./tmp/config
export STORAGE_DIR=./tmp/storage
export SERVER_PORT=8080

# 또는 .env 파일 생성
echo "CONFIG_DIR=./tmp/config" > .env
echo "STORAGE_DIR=./tmp/storage" >> .env
echo "SERVER_PORT=8080" >> .env
```

#### 포트 충돌
```bash
# 증상
bind: address already in use

# 해결책
# 1. 다른 포트 사용
export SERVER_PORT=8081

# 2. 기존 프로세스 종료
lsof -ti:8080 | xargs kill -9
```

### 2. 프록시 연결 문제

#### 업스트림 서버 연결 실패
```bash
# 진단
curl http://localhost:8080/healthz

# 응답에서 upstream 상태 확인
{
  "status": "unhealthy",
  "checks": {
    "upstream_connectivity": "failed"
  }
}

# 해결책
# 1. 네트워크 연결 확인
ping registry.npmjs.org

# 2. 프록시 설정 확인
proxyndctl config show

# 3. 방화벽 설정 확인
sudo ufw status
```

#### 클라이언트 설정 오류
```bash
# NPM 예시
# 증상: npm install 시 404 오류

# 해결책
npm config set registry http://localhost:8080/proxy/npm
npm config list  # 설정 확인
```

### 3. 캐시 관련 문제

#### 캐시 공간 부족
```bash
# 진단
proxyndctl cache size
du -sh $STORAGE_DIR

# 해결책
# 1. 오래된 캐시 정리
proxyndctl cache clear --older-than 30d

# 2. 특정 타입만 정리
proxyndctl cache clear --type npm

# 3. 캐시 크기 제한 설정 (global.yaml)
cache:
  max_size: "5GB"
  cleanup_threshold: "80%"
```

#### 캐시 권한 문제
```bash
# 증상
permission denied: /storage/cache

# 해결책
sudo chown -R $(whoami):$(whoami) $STORAGE_DIR
chmod -R 755 $STORAGE_DIR
```

### 4. 성능 문제

#### 응답 속도 저하
```bash
# 진단
# 1. 메트릭 확인
curl http://localhost:8080/metrics | grep response_time

# 2. 연결 풀 상태 확인
proxyndctl status --detailed

# 해결책
# 1. 연결 풀 크기 증가 (global.yaml)
connection_pool:
  max_connections: 100
  timeout: "30s"

# 2. 캐시 TTL 조정
cache:
  ttl: 7200  # 2시간으로 증가
```

#### 메모리 사용량 급증
```bash
# 진단
top -p $(pgrep proxynd)
ps aux | grep proxynd

# 해결책
# 1. 메모리 제한 설정
ulimit -m 2097152  # 2GB

# 2. GC 튜닝 (환경 변수)
export GOGC=100
export GOMEMLIMIT=2GiB
```

## 🔍 진단 도구

### 1. 내장 진단 명령어

```bash
# 전체 상태 확인
proxyndctl status

# 설정 검증
proxyndctl config validate

# 캐시 상태
proxyndctl cache list
proxyndctl cache size

# 특정 프록시 테스트
proxyndctl test --proxy npm
proxyndctl test all
```

### 2. 로그 분석

```bash
# 실시간 로그 모니터링
tail -f logs/proxynd.log | jq '.'

# 에러 로그 필터링
cat logs/proxynd.log | jq 'select(.level == "error")'

# 특정 시간대 로그
cat logs/proxynd.log | jq 'select(.timestamp > "2024-08-22T10:00:00Z")'
```

### 3. 메트릭 수집

```bash
# Prometheus 메트릭 확인
curl http://localhost:8080/metrics

# 주요 메트릭
# - proxynd_requests_total: 총 요청 수
# - proxynd_request_duration_seconds: 응답 시간
# - proxynd_cache_hits_total: 캐시 히트
# - proxynd_upstream_errors_total: 업스트림 오류
```

## 🚑 응급 조치

### 1. 서비스 재시작
```bash
# 개발 환경
make stop && make start

# 프로덕션 (systemd)
sudo systemctl restart proxynd

# Docker
docker restart proxynd
```

### 2. 캐시 초기화
```bash
# 전체 캐시 삭제
rm -rf $STORAGE_DIR/cache/*

# 또는
proxyndctl cache clear --all
```

### 3. 설정 초기화
```bash
# 기본 설정으로 복원
cp examples/config/*.yaml $CONFIG_DIR/

# 설정 검증
proxyndctl config validate
```

## 📊 모니터링 알림

문제가 발생하기 전에 예방하려면:

1. **[헬스체크 설정](health-checks.md)** - 정기적인 상태 확인
2. **[메트릭 수집](metrics-collection.md)** - Prometheus 연동
3. **[로깅 설정](logging-configuration.md)** - 구조화된 로그
4. **[성능 튜닝](performance-tuning.md)** - 최적화 가이드

## 🆘 추가 지원

문제가 지속되는 경우:

1. **이슈 생성**: [GitHub Issues](https://github.com/scriptonbasestar/proxynd/issues)
2. **로그 첨부**: 관련 에러 로그와 설정 파일
3. **환경 정보**: OS, Go 버전, ProxyND 버전
4. **재현 단계**: 문제 발생 과정 상세 기술

---

**관련 문서**:
- [모니터링 가이드](monitoring-guide.md)
- [헬스체크](health-checks.md)
- [성능 튜닝](performance-tuning.md)
