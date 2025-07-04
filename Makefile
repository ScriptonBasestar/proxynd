ENV=develop
DOCKER_REGISTRY=scriptonbasestar
VERSION?=latest

# Override only for specific targets that need these paths
PROD_STORAGE_DIR=~/tmp/storage/
PROD_CONFIG_DIR=~/tmp/config/

.PHONY: setup
setup:
	mkdir -p $(PROD_CONFIG_DIR)
	mkdir -p $(PROD_STORAGE_DIR)
	cp -r sample-conf/* $(PROD_CONFIG_DIR)

.PHONY: docker-build
docker-build:
	@echo "Building..."
	docker compose build --no-cache

.PHONY: docker-build-multiarch
docker-build-multiarch:
	@echo "Building multi-architecture images..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION}

.PHONY: docker-build-multiarch-push
docker-build-multiarch-push:
	@echo "Building and pushing multi-architecture images..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --push

.PHONY: docker-build-amd64
docker-build-amd64:
	@echo "Building AMD64 image..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --platforms linux/amd64 --load

.PHONY: docker-build-arm64
docker-build-arm64:
	@echo "Building ARM64 image..."
	./scripts/build-multiarch.sh --registry ${DOCKER_REGISTRY} --version ${VERSION} --platforms linux/arm64 --load

.PHONY: docker-push
docker-push:
	docker tag 'local_dev/proxynd' ${DOCKER_REGISTRY}/proxynd:latest
	docker push ${DOCKER_REGISTRY}/proxynd:latest

docker-run:
	@echo "Running..."
	docker compose up -d
	@echo "image name : proxynd"

docker-enter:
	@echo "Entering..."
	docker exec -it proxynd bash

.PHONY: local-buildlocal-

.PHONY: local-run
local-run:
	@echo "Running...?"
	./proxynd

.PHONY: dev-prepare
dev-prepare:
	@echo "Preparing development dependencies..."
	GOSUMDB=sum.golang.org go mod download
	GOSUMDB=sum.golang.org go mod tidy
	@echo "Installing air for hot reload..."
	@which air > /dev/null || GOSUMDB=sum.golang.org go install github.com/cosmtrek/air@latest
	@echo "Development dependencies ready!"

.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@mkdir -p ./tmp/storage
	@mkdir -p ./tmp/config
	@cp -r sample-conf/* ./tmp/config/
	@echo "Creating .env file..."
	@echo "CONFIG_DIR=./tmp/config" > .env
	@echo "STORAGE_DIR=./tmp/storage" >> .env
	@echo "SERVER_PORT=8080" >> .env
	@echo "Development environment ready!"
	@echo "Config dir: ./tmp/config"
	@echo "Storage dir: ./tmp/storage"
	@echo "Environment variables:"
	@cat .env

.PHONY: dev-run
dev-run: dev-setup
	@echo "Running with Air (hot reload)..."
	@which air > /dev/null || (echo "Air not found. Installing..." && go install github.com/cosmtrek/air@latest)
	air

.PHONY: dev-run-direct
dev-run-direct: dev-setup
	@echo "Running directly with go run..."
	CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage SERVER_PORT=8080 go run main.go

.PHONY: dev-test
dev-test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Organizing imports..."
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w -local proxynd .
	@echo "Code formatting complete!"

.PHONY: dev-teardown
dev-teardown:
	@echo "Cleaning up development environment..."
	@rm -rf ./tmp/
	@rm -f .env
	@echo "Development environment cleaned!"
