ARG GO_VERSION=1.22
ARG APP_BUILD_DIR=/go/src/app
ARG APP_RUN_DIR=/go/src/app

ARG GID=1000
ARG UID=1000
ARG APP_USER=user01
ARG APP_GROUP=group01

FROM golang:${GO_VERSION} as builder


# for debian
#RUN groupadd --gid $GID $APP_GROUP && useradd -m -l --uid $UID --gid $GID $APP_USER
#RUN mkdir -p $APP_DIR && chown -R $APP_USER:$APP_USER $APP_DIR
# for alpine
RUN addgroup -g $GID $APP_GROUP
RUN adduser -u $UID -G $APP_GROUP $APP_USER -h $APP_DIR -D
USER $APP_USER

WORKDIR $APP_BUILD_DIR

COPY . .
RUN go get -d -v ./...
#RUN go install -v ./...
RUN go build -o main


# run phase
FROM alpine:latest

ENV BASE_DIR=/storage
ENV CACHE_DIR=$BASE_DIR/cachedir
ENV MIRROR_DIR=$BASE_DIR/mirrordir
ENV CONFIG_DIR=$BASE_DIR/configdir

VOLUME $CACHE_DIR
VOLUME $MIRROR_DIR

EXPOSE 8080

RUN apk --no-cache add ca-certificates

WORKDIR $APP_RUN_DIR
COPY --from=builder $APP_BUILD_DIR/main .
CMD ["./main"]
