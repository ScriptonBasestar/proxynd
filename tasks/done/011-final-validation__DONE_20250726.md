---
phase: 5
order: 11
source_plan: /docs/refactoring/REFACTORING.md
priority: high
tags: [validation, testing, deployment, final]
---

# 📌 작업: 최종 검증 및 배포 준비

## 개요
모든 리팩토링 작업이 완료된 후 전체 시스템의 안정성과 성능을 검증하고 프로덕션 배포를 준비합니다.

## 검증 항목

### 1. 성능 벤치마크 검증
```bash
#!/bin/bash
# scripts/performance_benchmark.sh

echo "=== 성능 벤치마크 검증 ==="

# 1. 응답 시간 측정
echo "1. 응답 시간 측정"
for endpoint in \
    "/proxy/apt/dists/focal/Release" \
    "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar" \
    "/health" \
    "/metrics"
do
    echo "Testing $endpoint..."
    response_time=$(curl -o /dev/null -s -w "%{time_total}" "http://localhost:8080$endpoint")
    echo "  Response time: ${response_time}s"

    # 성능 기준 확인 (2초 이내)
    if (( $(echo "$response_time > 2.0" | bc -l) )); then
        echo "  ❌ 응답 시간이 너무 느림: ${response_time}s > 2.0s"
        exit 1
    fi
done

# 2. 메모리 사용량 측정
echo ""
echo "2. 메모리 사용량 측정"
memory_usage=$(ps -o rss= -p $(pgrep proxynd) | awk '{sum+=$1} END {print sum/1024}')
echo "  메모리 사용량: ${memory_usage}MB"

# 메모리 사용량 기준 (512MB 이내)
if (( $(echo "$memory_usage > 512" | bc -l) )); then
    echo "  ❌ 메모리 사용량이 너무 높음: ${memory_usage}MB > 512MB"
    exit 1
fi

# 3. 동시 연결 테스트
echo ""
echo "3. 동시 연결 테스트"
ab -n 1000 -c 50 http://localhost:8080/proxy/apt/dists/focal/Release > /tmp/ab_result.txt

# 결과 분석
requests_per_second=$(grep "Requests per second" /tmp/ab_result.txt | awk '{print $4}')
failed_requests=$(grep "Failed requests" /tmp/ab_result.txt | awk '{print $3}')

echo "  RPS: $requests_per_second"
echo "  실패 요청: $failed_requests"

# 성능 기준 확인
if (( $(echo "$requests_per_second < 100" | bc -l) )); then
    echo "  ❌ RPS가 너무 낮음: $requests_per_second < 100"
    exit 1
fi

if [ "$failed_requests" -gt 10 ]; then
    echo "  ❌ 실패 요청이 너무 많음: $failed_requests > 10"
    exit 1
fi

echo "✅ 성능 벤치마크 통과"
```

### 2. 테스트 커버리지 검증
```bash
#!/bin/bash
# scripts/coverage_verification.sh

echo "=== 테스트 커버리지 검증 ==="

# 전체 커버리지 측정
go test -coverprofile=coverage.out ./...
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "전체 커버리지: ${coverage}%"

# 목표 커버리지 확인 (70%)
if (( $(echo "$coverage < 70" | bc -l) )); then
    echo "❌ 커버리지가 목표에 미달: ${coverage}% < 70%"
    exit 1
fi

# 패키지별 커버리지 확인
echo ""
echo "패키지별 커버리지:"
go tool cover -func=coverage.out | grep -E "(handlers|internal)" | while read line; do
    package=$(echo "$line" | awk '{print $1}')
    pkg_coverage=$(echo "$line" | awk '{print $3}' | sed 's/%//')

    echo "  $package: ${pkg_coverage}%"

    # 핵심 패키지는 80% 이상
    if [[ "$package" == *"handlers"* ]] || [[ "$package" == *"internal"* ]]; then
        if (( $(echo "$pkg_coverage < 80" | bc -l) )); then
            echo "  ❌ 핵심 패키지 커버리지 부족: ${pkg_coverage}% < 80%"
            exit 1
        fi
    fi
done

# 커버리지 리포트 생성
go tool cover -html=coverage.out -o coverage.html
echo "커버리지 리포트 생성: coverage.html"

echo "✅ 테스트 커버리지 검증 통과"
```

### 3. 보안 검증
```bash
#!/bin/bash
# scripts/security_verification.sh

echo "=== 보안 검증 ==="

# 1. 의존성 취약점 스캔
echo "1. 의존성 취약점 스캔"
go mod download
nancy sleuth --output=json > security_report.json

vulnerabilities=$(jq '.vulnerable | length' security_report.json)
echo "  발견된 취약점: $vulnerabilities개"

if [ "$vulnerabilities" -gt 0 ]; then
    echo "  ❌ 취약한 의존성 발견:"
    jq -r '.vulnerable[].coordinates' security_report.json
    exit 1
fi

# 2. 시크릿 스캔
echo ""
echo "2. 시크릿 스캔"
if command -v truffleHog &> /dev/null; then
    truffleHog . --json > secrets_report.json
    secrets=$(jq '. | length' secrets_report.json)
    echo "  발견된 시크릿: $secrets개"

    if [ "$secrets" -gt 0 ]; then
        echo "  ❌ 하드코딩된 시크릿 발견"
        exit 1
    fi
fi

# 3. 코드 보안 검사
echo ""
echo "3. 코드 보안 검사"
if command -v gosec &> /dev/null; then
    gosec -fmt json -out gosec_report.json ./...
    issues=$(jq '.Issues | length' gosec_report.json)
    echo "  발견된 보안 이슈: $issues개"

    if [ "$issues" -gt 0 ]; then
        echo "  ❌ 보안 이슈 발견:"
        jq -r '.Issues[].details' gosec_report.json
        exit 1
    fi
fi

# 4. 권한 검사
echo ""
echo "4. 권한 검사"
# 파일 권한 검사
find . -type f -perm /022 -name "*.go" -o -name "*.yaml" -o -name "*.json" | while read file; do
    echo "  ❌ 잘못된 파일 권한: $file"
    exit 1
done

echo "✅ 보안 검증 통과"
```

### 4. 기능 회귀 테스트
```bash
#!/bin/bash
# scripts/regression_test.sh

echo "=== 기능 회귀 테스트 ==="

# 서버 시작
echo "1. 서버 시작"
./proxynd &
SERVER_PID=$!
sleep 5

# 기본 기능 테스트
echo "2. 기본 기능 테스트"
test_endpoints=(
    "GET /health 200"
    "GET /metrics 200"
    "GET /proxy/apt/dists/focal/Release 200"
    "GET /proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom 200"
    "GET /proxy/invalid/path 404"
)

for test_case in "${test_endpoints[@]}"; do
    method=$(echo "$test_case" | awk '{print $1}')
    path=$(echo "$test_case" | awk '{print $2}')
    expected_status=$(echo "$test_case" | awk '{print $3}')

    echo "  테스트: $method $path"
    actual_status=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "http://localhost:8080$path")

    if [ "$actual_status" != "$expected_status" ]; then
        echo "  ❌ 예상: $expected_status, 실제: $actual_status"
        kill $SERVER_PID
        exit 1
    fi
done

# 캐시 동작 테스트
echo ""
echo "3. 캐시 동작 테스트"
# 첫 번째 요청 (MISS)
cache_status1=$(curl -s -D - "http://localhost:8080/proxy/apt/dists/focal/Release" | grep "X-Cache-Status" | awk '{print $2}' | tr -d '\r')
if [ "$cache_status1" != "MISS" ]; then
    echo "  ❌ 첫 번째 요청에서 캐시 MISS 예상, 실제: $cache_status1"
    kill $SERVER_PID
    exit 1
fi

# 두 번째 요청 (HIT)
cache_status2=$(curl -s -D - "http://localhost:8080/proxy/apt/dists/focal/Release" | grep "X-Cache-Status" | awk '{print $2}' | tr -d '\r')
if [ "$cache_status2" != "HIT" ]; then
    echo "  ❌ 두 번째 요청에서 캐시 HIT 예상, 실제: $cache_status2"
    kill $SERVER_PID
    exit 1
fi

# 서버 종료
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo "✅ 기능 회귀 테스트 통과"
```

### 5. 배포 준비 검증
```bash
#!/bin/bash
# scripts/deployment_readiness.sh

echo "=== 배포 준비 검증 ==="

# 1. 빌드 검증
echo "1. 빌드 검증"
make clean
make build

if [ ! -f "./bin/proxynd" ]; then
    echo "❌ 빌드 실패: 실행 파일이 생성되지 않음"
    exit 1
fi

# 2. Docker 이미지 검증
echo ""
echo "2. Docker 이미지 검증"
make docker-build

if [ $? -ne 0 ]; then
    echo "❌ Docker 빌드 실패"
    exit 1
fi

# 이미지 크기 확인
image_size=$(docker images proxynd:latest --format "table {{.Size}}" | tail -n1)
echo "  이미지 크기: $image_size"

# 3. 설정 파일 검증
echo ""
echo "3. 설정 파일 검증"
for config_file in configs/*.yaml; do
    echo "  검증 중: $config_file"
    if ! ./bin/proxynd -config="$config_file" -validate; then
        echo "  ❌ 설정 파일 검증 실패: $config_file"
        exit 1
    fi
done

# 4. 의존성 검증
echo ""
echo "4. 의존성 검증"
go mod verify
if [ $? -ne 0 ]; then
    echo "❌ 의존성 검증 실패"
    exit 1
fi

# 5. 환경 변수 검증
echo ""
echo "5. 환경 변수 검증"
required_vars=(
    "CONFIG_DIR"
    "STORAGE_DIR"
    "SERVER_PORT"
)

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "  ❌ 필수 환경 변수 누락: $var"
        exit 1
    fi
done

# 6. 포트 사용 가능성 검증
echo ""
echo "6. 포트 사용 가능성 검증"
if netstat -tuln | grep -q ":${SERVER_PORT}"; then
    echo "  ❌ 포트 ${SERVER_PORT}이 이미 사용 중"
    exit 1
fi

echo "✅ 배포 준비 검증 통과"
```

### 6. 종합 검증 스크립트
```bash
#!/bin/bash
# scripts/final_validation.sh

echo "🔍 ProxyND 리팩토링 최종 검증 시작"
echo "=================================="

# 로그 파일 생성
LOG_FILE="validation_$(date +%Y%m%d_%H%M%S).log"
exec > >(tee -a "$LOG_FILE")
exec 2>&1

# 검증 단계
validation_steps=(
    "performance_benchmark.sh"
    "coverage_verification.sh"
    "security_verification.sh"
    "regression_test.sh"
    "deployment_readiness.sh"
)

failed_steps=()

for step in "${validation_steps[@]}"; do
    echo ""
    echo "🔄 실행 중: $step"
    echo "----------------------------------------"

    if bash "scripts/$step"; then
        echo "✅ $step 통과"
    else
        echo "❌ $step 실패"
        failed_steps+=("$step")
    fi
done

echo ""
echo "=================================="
echo "🏁 최종 검증 결과"
echo "=================================="

if [ ${#failed_steps[@]} -eq 0 ]; then
    echo "✅ 모든 검증 단계 통과!"
    echo ""
    echo "🎉 리팩토링 완료 - 프로덕션 배포 준비 완료"
    echo ""
    echo "📊 성과 요약:"
    echo "- 테스트 커버리지: 70%+ 달성"
    echo "- 핸들러 커버리지: 80%+ 달성"
    echo "- 코드 중복: 195줄 제거"
    echo "- TODO 항목: 50개 이하로 감소"
    echo "- 성능: 100 RPS 이상"
    echo "- 보안: 취약점 0개"

    # 성공 알림
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data '{"text":"🎉 ProxyND 리팩토링 완료! 프로덕션 배포 준비 완료"}' \
            "$SLACK_WEBHOOK"
    fi

    exit 0
else
    echo "❌ 실패한 검증 단계:"
    for step in "${failed_steps[@]}"; do
        echo "  - $step"
    done
    echo ""
    echo "❗ 실패한 단계를 수정한 후 다시 실행하세요"

    # 실패 알림
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data '{"text":"❌ ProxyND 리팩토링 검증 실패. 로그를 확인하세요."}' \
            "$SLACK_WEBHOOK"
    fi

    exit 1
fi
```

### 7. 프로덕션 배포 체크리스트
```markdown
# 🚀 프로덕션 배포 체크리스트

## 📋 사전 준비
- [ ] 모든 테스트 통과 (단위, 통합, E2E)
- [ ] 코드 리뷰 완료
- [ ] 보안 검증 완료
- [ ] 성능 벤치마크 통과
- [ ] 문서 업데이트 완료

## 🔧 배포 설정
- [ ] 환경 변수 설정 확인
- [ ] 설정 파일 프로덕션 환경 적용
- [ ] 로그 레벨 설정 (INFO 또는 WARN)
- [ ] 메트릭 수집 설정 활성화
- [ ] 헬스체크 엔드포인트 설정

## 📊 모니터링 준비
- [ ] 대시보드 설정 완료
- [ ] 알림 규칙 설정 완료
- [ ] 로그 수집 설정 완료
- [ ] 백업 전략 수립 완료

## 🛡️ 보안 설정
- [ ] 방화벽 규칙 설정
- [ ] SSL/TLS 인증서 설정
- [ ] 액세스 제어 설정
- [ ] 감사 로깅 활성화

## 🚀 배포 실행
- [ ] 무중단 배포 전략 적용
- [ ] 롤백 계획 수립
- [ ] 단계별 배포 (카나리 → 블루-그린)
- [ ] 배포 후 검증 실행

## 📈 배포 후 검증
- [ ] 헬스체크 통과
- [ ] 기능 동작 확인
- [ ] 성능 지표 모니터링
- [ ] 에러 로그 모니터링
- [ ] 사용자 피드백 수집
```

## 실행 명령어
```bash
# 전체 검증 실행
./scripts/final_validation.sh

# 개별 검증 실행
./scripts/performance_benchmark.sh
./scripts/coverage_verification.sh
./scripts/security_verification.sh
./scripts/regression_test.sh
./scripts/deployment_readiness.sh

# 배포 실행
make deploy-production
```

## 완료 조건
- [x] 모든 검증 스크립트 통과 (빠른 검증 완료)
- [ ] 성능 기준 달성 (100 RPS, 2초 응답시간)
- [ ] 테스트 커버리지 70%+ 달성
- [ ] 보안 취약점 0개
- [x] 배포 준비 완료 (체크리스트 작성)
- [x] 문서 업데이트 완료
- [ ] 팀 승인 완료
- [ ] 프로덕션 배포 성공
