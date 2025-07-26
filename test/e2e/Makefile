# E2E Test Makefile

.PHONY: up down test test-all clean logs help

# Docker Compose 파일 경로
COMPOSE_FILE := ../docker-compose.e2e.yml
PROJECT_NAME := proxynd-e2e

# 기본 타겟
help:
	@echo "ProxyND E2E Test Commands:"
	@echo ""
	@echo "  up          - Start E2E test environment"
	@echo "  down        - Stop E2E test environment"
	@echo "  test        - Run E2E tests"
	@echo "  test-all    - Run all E2E tests including client-specific tests"
	@echo "  clean       - Clean up E2E environment and volumes"
	@echo "  logs        - Show logs from all services"
	@echo "  logs-follow - Follow logs from all services"
	@echo "  status      - Show status of all services"
	@echo "  shell-*     - Open shell in specific container"
	@echo "  help        - Show this help message"

# 환경 시작
up:
	@echo "🚀 Starting E2E test environment..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ E2E environment is ready!"
	@echo ""
	@echo "Services available:"
	@echo "  - ProxyND: http://localhost:8080"
	@echo "  - MinIO: http://localhost:9001 (admin/admin123)"
	@echo "  - Upstream: http://localhost:8081"

# 환경 중지
down:
	@echo "🛑 Stopping E2E test environment..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down

# E2E 테스트 실행
test:
	@echo "🧪 Running E2E tests..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) run --rm test-runner

# 모든 E2E 테스트 실행 (클라이언트별 테스트 포함)
test-all: test test-npm test-pip test-apt test-docker

# NPM 클라이언트 테스트
test-npm:
	@echo "📦 Running NPM client tests..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec node-client /scripts/test-npm.sh

# PyPI 클라이언트 테스트
test-pip:
	@echo "🐍 Running PyPI client tests..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec python-client /scripts/test-pip.sh

# APT 클라이언트 테스트
test-apt:
	@echo "📦 Running APT client tests..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec ubuntu-client /scripts/test-apt.sh

# Docker 클라이언트 테스트
test-docker:
	@echo "🐳 Running Docker client tests..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec docker-client /scripts/test-docker.sh

# 로그 보기
logs:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs

# 실시간 로그 따라가기
logs-follow:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f

# ProxyND 로그만 보기
logs-proxynd:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs proxynd

# 서비스 상태 확인
status:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) ps

# 컨테이너 셸 접근
shell-proxynd:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec proxynd sh

shell-ubuntu:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec ubuntu-client bash

shell-node:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec node-client sh

shell-python:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec python-client sh

shell-docker:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) exec docker-client sh

# 정리
clean:
	@echo "🧹 Cleaning up E2E environment..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down -v
	docker system prune -f --filter label=com.docker.compose.project=$(PROJECT_NAME)

# 빠른 테스트 (환경 시작 + 테스트 + 정리)
quick-test: up test down

# 개발용 - 환경 재시작
restart: down up

# 헬스체크
health:
	@echo "🏥 Checking service health..."
	@curl -sf http://localhost:8080/healthz || echo "❌ ProxyND health check failed"
	@curl -sf http://localhost:9000/minio/health/live || echo "❌ MinIO health check failed"
	@curl -sf http://localhost:8081/health || echo "❌ Upstream health check failed"

# 서비스별 개별 시작
up-proxynd:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d proxynd

up-minio:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d minio

up-upstream:
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d nginx-upstream

# 볼륨 정보
volumes:
	docker volume ls --filter label=com.docker.compose.project=$(PROJECT_NAME)

# 네트워크 정보
network:
	docker network ls --filter label=com.docker.compose.project=$(PROJECT_NAME)
