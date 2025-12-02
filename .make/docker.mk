# Makefile.docker.mk - Docker and Container Management
# All Docker-related operations and container management

# ==============================================================================
# Docker Configuration
# ==============================================================================


# ==============================================================================
# Docker Build Targets
# ==============================================================================

.PHONY: docker-build docker-build-multiarch docker-build-multiarch-push
.PHONY: docker-build-amd64 docker-build-arm64 docker-push docker-run docker-enter

docker-build: ## build docker image locally
	@echo "Building Docker image..."
	docker compose build --no-cache
	@echo "✅ Docker build complete!"

docker-build-multiarch: ## build multi-architecture images
	@echo "Building multi-architecture images..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION}
	@echo "✅ Multi-architecture build complete!"

docker-build-multiarch-push: ## build and push multi-architecture images
	@echo "Building and pushing multi-architecture images..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --push
	@echo "✅ Multi-architecture build and push complete!"

docker-build-amd64: ## build AMD64 image only
	@echo "Building AMD64 image..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --platforms linux/amd64 --load
	@echo "✅ AMD64 build complete!"

docker-build-arm64: ## build ARM64 image only
	@echo "Building ARM64 image..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --platforms linux/arm64 --load
	@echo "✅ ARM64 build complete!"

# ==============================================================================
# Docker Registry Operations
# ==============================================================================

docker-push: ## push docker image to registry
	@echo "Pushing to registry..."
	docker tag 'local_dev/proxynd' ${DOCKER_REGISTRY}/proxynd:latest
	docker push ${DOCKER_REGISTRY}/proxynd:latest
	@echo "✅ Push complete!"

# ==============================================================================
# Docker Runtime Operations
# ==============================================================================

docker-run: ## run docker container with docker-compose
	@echo "Starting ProxyND container..."
	docker compose up -d
	@echo "✅ Container started!"
	@echo "Container name: proxynd"
	@echo ""
	@echo "Useful commands:"
	@echo "  make docker-enter    # Enter running container"
	@echo "  docker logs proxynd  # View container logs"
	@echo "  docker compose down  # Stop container"

docker-enter: ## enter running docker container
	@echo "Entering ProxyND container..."
	docker exec -it proxynd bash

# ==============================================================================
# Docker Cleanup
# ==============================================================================

.PHONY: docker-clean docker-clean-all docker-clean-images docker-clean-containers

docker-clean: ## clean docker build cache
	@echo "Cleaning Docker build cache..."
	docker system prune -f
	@echo "✅ Docker cache cleaned!"

docker-clean-images: ## clean unused docker images
	@echo "Cleaning unused Docker images..."
	docker image prune -f
	@echo "✅ Unused images cleaned!"

docker-clean-containers: ## clean stopped containers
	@echo "Cleaning stopped containers..."
	docker container prune -f
	@echo "✅ Stopped containers cleaned!"

docker-clean-all: ## clean all docker resources
	@echo "Performing comprehensive Docker cleanup..."
	docker system prune -a -f --volumes
	@echo "⚠️  All Docker resources cleaned (images, containers, volumes, networks)!"

# ==============================================================================
# Docker Development
# ==============================================================================

.PHONY: docker-dev docker-dev-logs docker-dev-restart

docker-dev: ## start development environment with docker
	@echo "Starting development environment..."
	docker compose -f docker-compose.yml up -d
	@echo "✅ Development environment started!"
	@echo ""
	@echo "Services:"
	@docker compose ps

docker-dev-logs: ## view logs from development containers
	@echo "Viewing development container logs..."
	docker compose logs -f

docker-dev-restart: ## restart development containers
	@echo "Restarting development containers..."
	docker compose restart
	@echo "✅ Development containers restarted!"

# ==============================================================================
# Docker Testing
# ==============================================================================

.PHONY: docker-test docker-test-integration

docker-test: ## run tests in docker container
	@echo "Running tests in Docker..."
	docker compose exec proxynd make test-unit
	@echo "✅ Docker tests complete!"

docker-test-integration: ## run integration tests with docker
	@echo "Running integration tests with Docker..."
	./tests/e2e/scripts/test-docker.sh
	@echo "✅ Docker integration tests complete!"

# ==============================================================================
# Docker Information
# ==============================================================================

.PHONY: docker-info docker-status

docker-info: ## show docker system information
	@echo "=== Docker System Information ==="
	docker version
	@echo ""
	@echo "=== Docker Images ==="
	docker images | grep -E "(proxynd|scriptonbasestar)" || echo "No ProxyND images found"
	@echo ""
	@echo "=== Docker Containers ==="
	docker ps -a | grep proxynd || echo "No ProxyND containers found"

docker-status: ## show status of docker containers
	@echo "=== Docker Container Status ==="
	docker compose ps
	@echo ""
	@echo "=== Resource Usage ==="
	docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" | head -n 10
