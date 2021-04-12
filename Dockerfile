FROM go:1.15-alpine

ENV BASE_DIR=/storage
ENV CACHE_DIR=$BASE_DIR/cachedir
ENV MIRROR_DIR=$BASE_DIR/mirrordir
ENV CONFIG_DIR=$BASE_DIR/configdir

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

VOLUME $CACHE_DIR
VOLUME $MIRROR_DIR
EXPOSE 8080

CMD ["app"]
