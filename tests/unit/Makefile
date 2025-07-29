# Unit Test Makefile

.PHONY: test test-verbose test-short test-race test-cache test-middleware test-router test-helpers benchmark coverage clean

# 전체 단위 테스트 실행
test:
	go test -v -timeout 30s

# 자세한 출력으로 테스트
test-verbose:
	go test -v -timeout 30s -count=1

# 빠른 테스트 (짧은 테스트만)
test-short:
	go test -v -short -timeout 10s

# 레이스 컨디션 체크와 함께 테스트
test-race:
	go test -v -race -timeout 60s

# 개별 테스트 모듈별 실행
test-cache:
	go test -v -run "TestFileSystemBackend|TestCacheManager|TestCacheEviction" -timeout 30s

test-middleware:
	go test -v -run "TestIPFilterMiddleware|TestBasicAuthMiddleware|TestPermissionMiddleware|TestPackageFilterMiddleware|TestSecurityMiddleware" -timeout 30s

test-router:
	go test -v -run "TestHealthRouter|TestProxyRouter|TestRouterParameterExtraction|TestRouterMiddlewareChain" -timeout 30s

test-helpers:
	go test -v -run "TestYAMLHelper|TestURLHelper" -timeout 30s

# 벤치마크 테스트
benchmark:
	go test -v -bench=. -timeout 60s

# 테스트 커버리지
coverage:
	go test -v -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 상세 커버리지 (함수별)
coverage-detail:
	go test -v -cover -coverprofile=coverage.out
	go tool cover -func=coverage.out

# 테스트 환경 정리
clean:
	@echo "Cleaning up test environment..."
	@rm -f coverage.out coverage.html
	@rm -rf /tmp/proxynd-*
	@echo "Test environment cleaned"

# 특정 테스트 실행 (TEST 변수 사용)
# 사용법: make test-specific TEST=TestCacheManager
test-specific:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-specific TEST=TestName"; \
		exit 1; \
	fi
	go test -v -run "$(TEST)" -timeout 30s

# 테스트 실패 시 상세 정보 출력
test-debug:
	go test -v -timeout 30s -args -test.v

# CI/CD용 테스트 (모든 검증 포함)
ci-test: test-race coverage
	@echo "All CI tests completed successfully"

# 성능 프로파일링
profile-cpu:
	go test -v -cpuprofile=cpu.prof -bench=.
	go tool pprof cpu.prof

profile-mem:
	go test -v -memprofile=mem.prof -bench=.
	go tool pprof mem.prof

# 테스트 데이터 생성 (필요한 경우)
test-data:
	@echo "Generating test data..."
	@mkdir -p /tmp/proxynd-test-data
	@echo "name: test-config" > /tmp/proxynd-test-data/test.yaml
	@echo "version: 1.0.0" >> /tmp/proxynd-test-data/test.yaml
	@echo "Test data generated in /tmp/proxynd-test-data/"

# 병렬 테스트 실행
test-parallel:
	go test -v -parallel 4 -timeout 60s

# 테스트 결과를 JUnit XML 형식으로 출력 (CI/CD용)
test-junit:
	go test -v -timeout 30s 2>&1 | go-junit-report > test-results.xml

# 도움말
help:
	@echo "Available targets:"
	@echo "  test              - Run all unit tests"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  test-short        - Run short tests only"
	@echo "  test-race         - Run tests with race condition detection"
	@echo "  test-cache        - Run cache-related tests only"
	@echo "  test-middleware   - Run middleware tests only"
	@echo "  test-router       - Run router tests only"
	@echo "  test-helpers      - Run helper tests only"
	@echo "  test-specific     - Run specific test (use TEST=TestName)"
	@echo "  benchmark         - Run benchmark tests"
	@echo "  coverage          - Generate test coverage report"
	@echo "  coverage-detail   - Show detailed coverage by function"
	@echo "  test-parallel     - Run tests in parallel"
	@echo "  ci-test           - Run full CI test suite"
	@echo "  clean             - Clean up test artifacts"
	@echo "  help              - Show this help message"
