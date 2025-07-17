# 성능 테스트 가이드

## 📊 개요

ProxyND의 성능 벤치마크 테스트 시스템은 다음 영역을 커버합니다:

- **프록시 요청 처리**: 다양한 크기의 패키지 요청 성능
- **캐시 시스템**: 파일시스템 캐시의 읽기/쓰기 성능
- **미들웨어**: 보안, 로깅, 메트릭 수집 성능
- **설정 관리**: 설정 로딩 및 파싱 성능

## 🚀 빠른 시작

### 기본 벤치마크 실행
```bash
# 모든 벤치마크 실행
make test-benchmark

# 특정 영역 벤치마크
make test-benchmark-proxy      # 프록시 성능
make test-benchmark-cache      # 캐시 성능
make test-benchmark-middleware # 미들웨어 성능
make test-benchmark-config     # 설정 성능
```

### 상세 벤치마크 스크립트
```bash
# 기본 실행
./scripts/run_benchmarks.sh

# 10초간 실행
./scripts/run_benchmarks.sh --all -t 10s

# 프로파일링과 함께 실행
./scripts/run_benchmarks.sh --proxy -p

# 이전 결과와 비교
./scripts/run_benchmarks.sh -c reports/benchmarks/baseline.txt
```

## 📈 벤치마크 카테고리

### 1. 프록시 성능 벤치마크

#### 패키지 크기별 성능
```bash
# 작은 패키지 (1KB)
go test -bench=BenchmarkProxyRequest_SmallPackage ./tests/benchmark/

# 중간 패키지 (1MB)
go test -bench=BenchmarkProxyRequest_MediumPackage ./tests/benchmark/

# 큰 패키지 (10MB)
go test -bench=BenchmarkProxyRequest_LargePackage ./tests/benchmark/
```

#### 프록시 타입별 성능
```bash
# APT 프록시
go test -bench=BenchmarkProxyRequest_APT ./tests/benchmark/

# Maven 프록시
go test -bench=BenchmarkProxyRequest_Maven ./tests/benchmark/

# NPM 프록시 (기본)
go test -bench=BenchmarkProxyRequest ./tests/benchmark/
```

#### 동시 요청 성능
```bash
# 동시 요청 처리
go test -bench=BenchmarkProxyRequest_ConcurrentRequests ./tests/benchmark/
```

### 2. 캐시 성능 벤치마크

#### 파일시스템 캐시
```bash
# 데이터 크기별 성능
go test -bench=BenchmarkCacheFileSystem ./tests/benchmark/

# 동시 접근 성능
go test -bench=BenchmarkCacheConcurrency ./tests/benchmark/

# 캐시 만료 처리
go test -bench=BenchmarkCacheExpiration ./tests/benchmark/
```

#### 캐시 작업별 성능
```bash
# 기본 캐시 작업 (Set/Get/Exists)
go test -bench=BenchmarkCacheOperations ./tests/benchmark/

# 메모리 효율성
go test -bench=BenchmarkCacheMemoryEfficiency ./tests/benchmark/
```

### 3. 미들웨어 성능 벤치마크

#### 보안 미들웨어
```bash
# 보안 미들웨어 전체
go test -bench=BenchmarkSecurityMiddleware ./tests/benchmark/

# 속도 제한
go test -bench=BenchmarkRateLimitMiddleware ./tests/benchmark/

# CORS 처리
go test -bench=BenchmarkCORSMiddleware ./tests/benchmark/
```

#### 로깅 및 메트릭
```bash
# 로깅 미들웨어
go test -bench=BenchmarkLoggingMiddleware ./tests/benchmark/

# 메트릭 수집
go test -bench=BenchmarkMetricsMiddleware ./tests/benchmark/
```

#### 압축 및 인증
```bash
# 압축 미들웨어
go test -bench=BenchmarkCompressionMiddleware ./tests/benchmark/

# 인증 미들웨어
go test -bench=BenchmarkAuthMiddleware ./tests/benchmark/
```

### 4. 설정 관리 벤치마크

#### 설정 로딩
```bash
# 설정 타입별 로딩 성능
go test -bench=BenchmarkConfigLoading ./tests/benchmark/

# 설정 파싱 성능
go test -bench=BenchmarkConfigParsing ./tests/benchmark/
```

#### 고급 설정 기능
```bash
# 핫 리로드 성능
go test -bench=BenchmarkConfigHotReload ./tests/benchmark/

# 동시 접근 성능
go test -bench=BenchmarkConfigConcurrency ./tests/benchmark/
```

## 📊 성능 기준

### 성능 목표

| 영역 | 메트릭 | 목표 값 | 측정 방법 |
|------|--------|---------|-----------|
| 프록시 요청 | 처리량 | > 1000 req/sec | BenchmarkProxyRequest |
| 캐시 읽기 | 속도 | < 1ms | BenchmarkCacheFileSystem |
| 캐시 쓰기 | 속도 | < 10ms | BenchmarkCacheFileSystem |
| 미들웨어 | 오버헤드 | < 100μs | BenchmarkMiddlewareStack |
| 설정 로딩 | 시간 | < 50ms | BenchmarkConfigLoading |

### 성능 회귀 감지

#### 기준선 생성
```bash
# 기준선 벤치마크 실행 및 저장
./scripts/run_benchmarks.sh --all -t 30s > reports/benchmarks/baseline.txt
```

#### 회귀 검사
```bash
# 현재 성능과 기준선 비교
./scripts/run_benchmarks.sh --all -c reports/benchmarks/baseline.txt
```

#### 자동화된 성능 검사
```bash
# CI/CD에서 사용할 수 있는 성능 검사
./scripts/run_benchmarks.sh --all -t 10s | grep -E "(PASS|FAIL|slower|faster)"
```

## 🔧 프로파일링

### CPU 프로파일링
```bash
# CPU 프로파일과 함께 벤치마크 실행
./scripts/run_benchmarks.sh --proxy -p

# 프로파일 분석
go tool pprof reports/benchmarks/profiles_TIMESTAMP/cpu.prof
```

### 메모리 프로파일링
```bash
# 메모리 프로파일 분석
go tool pprof reports/benchmarks/profiles_TIMESTAMP/mem.prof
```

### 프로파일링 명령어 예시
```bash
# 웹 브라우저에서 프로파일 보기
go tool pprof -http=:8080 cpu.prof

# 텍스트 모드에서 상위 10개 함수 보기
go tool pprof -text -nodecount=10 cpu.prof

# 호출 그래프 생성
go tool pprof -svg cpu.prof > cpu_profile.svg
```

## 📋 벤치마크 해석

### 벤치마크 결과 읽기
```
BenchmarkProxyRequest_SmallPackage-8    1000    1234567 ns/op    4096 B/op    64 allocs/op
```

- `BenchmarkProxyRequest_SmallPackage`: 벤치마크 이름
- `-8`: GOMAXPROCS 값
- `1000`: 실행 횟수
- `1234567 ns/op`: 작업당 나노초
- `4096 B/op`: 작업당 바이트 할당
- `64 allocs/op`: 작업당 메모리 할당 횟수

### 성능 지표 해석

#### 처리량 (Throughput)
```bash
# ns/op에서 req/sec 계산
# req/sec = 1,000,000,000 / ns_per_op
```

#### 메모리 효율성
- `B/op`: 낮을수록 좋음
- `allocs/op`: 낮을수록 좋음 (GC 부담 감소)

#### 지연시간 (Latency)
- `ns/op`: 낮을수록 좋음
- 일관성도 중요 (표준편차 확인)

## 🎯 성능 최적화 가이드

### 1. 프록시 성능 최적화

#### HTTP 클라이언트 최적화
```go
// 커넥션 풀 설정
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

#### 버퍼링 최적화
```go
// 큰 패키지를 위한 스트리밍
io.CopyBuffer(dst, src, make([]byte, 32*1024))
```

### 2. 캐시 성능 최적화

#### 파일시스템 최적화
- SSD 사용 권장
- 적절한 파일시스템 선택 (ext4, xfs)
- 캐시 디렉토리 분산

#### 메모리 캐시 추가
```go
// LRU 캐시로 자주 사용되는 항목 메모리에 보관
cache := NewLRUCache(1000)
```

### 3. 미들웨어 성능 최적화

#### 미들웨어 순서 최적화
```go
// 빠른 미들웨어를 앞에 배치
app.Use(rateLimit)      // 빠름
app.Use(compression)    // 중간
app.Use(logging)        // 느림
```

#### 조건부 미들웨어
```go
// 개발 환경에서만 상세 로깅
if config.Environment == "development" {
    app.Use(detailedLogging)
}
```

## 📈 연속적인 성능 모니터링

### CI/CD 통합
```yaml
# .github/workflows/benchmark.yml
- name: Run Benchmarks
  run: |
    ./scripts/run_benchmarks.sh --all -t 30s > benchmark_results.txt
    ./scripts/check_performance_regression.sh benchmark_results.txt
```

### 성능 대시보드
- Grafana와 Prometheus 연동
- 핵심 성능 지표 모니터링
- 알림 설정

### 정기 성능 리뷰
- 주간 성능 리포트
- 성능 회귀 분석
- 최적화 계획 수립

## 🔍 문제 해결

### 일반적인 성능 문제

#### 메모리 누수
```bash
# 메모리 사용량 모니터링
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

#### CPU 병목
```bash
# CPU 프로파일링
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

#### 고루틴 누수
```bash
# 고루틴 덤프
kill -SIGQUIT $PID
```

### 성능 저하 원인 분석

1. **네트워크 지연**: 업스트림 서버 응답 시간 확인
2. **디스크 I/O**: 캐시 디스크 성능 확인
3. **메모리 부족**: 시스템 메모리 사용률 확인
4. **CPU 사용률**: 높은 CPU 사용률 시 병목 지점 분석

## 📚 참고 자료

### Go 벤치마킹
- [Go Testing Package](https://golang.org/pkg/testing/)
- [Benchmarking in Go](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

### 프로파일링 도구
- [pprof User Guide](https://github.com/google/pprof)
- [Go Profiling](https://golang.org/doc/diagnostics.html)

### 성능 최적화
- [High Performance Go](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)
- [Go Performance Tips](https://github.com/golang/go/wiki/Performance)

---

**마지막 업데이트**: 2024년 7월 17일  
**작성자**: Claude Code Assistant