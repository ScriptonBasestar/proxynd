FROM golang:alpine

ENV BASE_DIR=/storage
ENV CACHE_DIR=/cachedir
ENV MIRROR_DIR=/mirrordir

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]