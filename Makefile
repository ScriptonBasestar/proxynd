ENV=develop
#DOCKER_REGISTRY=scripton-io
DOCKER_REGISTRY=archmagece

setup:
	mkdir -p ~/tmp/config
	mkdir -p ~/tmp/storage
	cp -r sample-conf/* ~/tmp/config/.

docker-build:
	@echo "Building..."
	docker compose build --no-cache

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

.PHONY: build
