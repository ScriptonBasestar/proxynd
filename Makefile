ENV=develop
DOCKER_REGISTRY=scriptonbasestar

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

.PHONY: docker-push
docker-push:
	docker tag 'local_dev/proxynd' ${DOCKER_REGISTRY}/proxynd:latest
	docker push ${DOCKER_REGISTRY}/proxynd:latest

.PHONY: docker-run
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
