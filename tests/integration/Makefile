# Integration Test Makefile

.PHONY: test test-verbose test-short test-race clean setup

# 기본 테스트 실행
test:
	go test -v -timeout 30s

# 자세한 출력으로 테스트 실행
test-verbose:
	go test -v -timeout 30s -count=1

# 빠른 테스트 (짧은 테스트만)
test-short:
	go test -v -short -timeout 10s

# 레이스 컨디션 체크와 함께 테스트
test-race:
	go test -v -race -timeout 60s

# 벤치마크 테스트
benchmark:
	go test -v -bench=. -timeout 60s

# 테스트 커버리지
coverage:
	go test -v -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# 테스트 환경 설정
setup:
	@echo "Setting up test environment..."
	@mkdir -p /tmp/proxynd-test-cache
	@mkdir -p /tmp/proxynd-test-logs
	@echo "Test environment setup complete"

# 테스트 환경 정리
clean:
	@echo "Cleaning up test environment..."
	@rm -rf /tmp/proxynd-test-cache
	@rm -rf /tmp/proxynd-test-logs
	@rm -f coverage.out coverage.html
	@echo "Test environment cleaned"

# 전체 테스트 스위트 실행
full-test: clean setup test

# CI/CD용 테스트
ci-test: setup test-race coverage