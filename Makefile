ENV=develop
DOCKER_REGISTRY=scriptonbasestar
VERSION?=latest

.EXPORT_ALL_VARIABLES:
STORAGE_DIR=~/tmp/storage/
CONFIG_DIR=~/tmp/config/

.PHONY: setup
setup:
	mkdir -p ~/tmp/config
	mkdir -p ~/tmp/storage
	cp -r sample-conf/* ~/tmp/config/.

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
	@echo "Preparing..."
	go mod download
	go mod vendor
	go mod tidy

.PHONY: dev-setup
dev-setup:
	@echo "Setting up..."
	@mkdir -p ~/tmp/config
	@cp -r sample-conf/* ~/tmp/config/.

.PHONY: dev-run
dev-run:
	@echo "Running..."
	go run main.go

.PHONY: dev-test
dev-test:
	@echo "Testing..."
	go test -v ./...
