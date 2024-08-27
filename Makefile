ENV=develop
#DOCKER_REGISTRY=scripton-io
DOCKER_REGISTRY=archmagece

setup:
	cp -r conf/ tmpconf/

docker-build:
	@echo "Building..."
	docker compose build --no-cache

docker-push:
	docker tag deproxy ${DOCKER_REGISTRY}/deproxy:latest .
	docker push ${DOCKER_REGISTRY}/deproxy:latest

docker-run:
	@echo "Running..."
	docker compose up -d
	@echo "image name : deproxy"

docker-enter:
	@echo "Entering..."
	docker exec -it deproxy bash

.PHONY: build
