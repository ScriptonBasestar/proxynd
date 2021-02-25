FROM golang:alpine

ENV BASE_DIR=/storage
ENV CACHE_DIR=$BASE_DIR/cachedir
ENV MIRROR_DIR=$BASE_DIR/mirrordir
ENV CONFIG_DIR=$BASE_DIR/configdir

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]