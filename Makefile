ENV=develop
#DOCKER_REGISTRY=scripton-io
DOCKER_REGISTRY=archmagece


docker-build:
	@echo "Building..."
	docker build -t deproxy .

docker-push:
	docker tag deproxy ${DOCKER_REGISTRY}/deproxy:latest .
	docker push ${DOCKER_REGISTRY}/deproxy:latest

docker-run:
	@echo "Running..."
	docker run -it -p 8080:8080 --rm deproxy
	@echo "image name : deproxy"

test-run:
	@echo "Running..."
	docker run -it --rm golang bash
#	export MYGID=1000
#	export MYUID=1000
#	export APP_USER=user01
#	export APP_GROUP=group01
#	RUN groupadd -g ${MYGID} ${APP_GROUP}
#	RUN useradd -g $MYGID -o $APP_USER -u $MYUID -d /home/$APP_USER -s /bin/bash -m $APP_USER

.PHONY: build
