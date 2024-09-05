ARG GO_VERSION=1.22

FROM golang:${GO_VERSION} AS builder

ARG APP_BUILD_DIR=/go/src/app
ARG APP_RUN_DIR=/go/src/app
#ARG MYGID=1001
#ARG MYUID=1001
#ARG APP_USER=user01
#ARG APP_GROUP=group01
#ENV MYGID=${MYGID}
#ENV MYUID=${MYUID}
#ENV APP_USER=${APP_USER}
#ENV APP_GROUP=${APP_GROUP}

# for debian
#RUN groupadd --gid $GID $APP_GROUP && useradd -m -l --uid $UID --gid $GID $APP_USER
#RUN mkdir -p $APP_DIR && chown -R $APP_USER:$APP_USER $APP_DIR
# for alpine
#RUN echo "1111111111111 ${MYGID} ${APP_GROUP}"
#RUN groupadd -g ${MYGID} -o ${APP_GROUP}
#RUN useradd -g $MYGID -o $APP_USER -u $MYUID -d /home/$APP_USER
#USER $APP_USER

WORKDIR $APP_BUILD_DIR

#COPY . .
COPY . $APP_BUILD_DIR
RUN go get -d -v ./...
RUN go mod download
RUN go build -o main


# run phase
FROM golang:${GO_VERSION}

ARG APP_BUILD_DIR=/go/src/app
ARG APP_RUN_DIR=/go/src/app

ENV STORAGE_DIR=/storage
#ENV CACHE_DIR=$STORAGE_DIR/cachedir
#ENV MIRROR_DIR=$STORAGE_DIR/mirrordir
#ENV CONFIG_DIR=$STORAGE_DIR/configdir
ENV CONFIG_DIR=/config

#VOLUME $CACHE_DIR
#VOLUME $MIRROR_DIR

EXPOSE 8080

RUN #apk --no-cache add ca-certificates

WORKDIR $APP_RUN_DIR
COPY --from=builder $APP_BUILD_DIR/main $APP_RUN_DIR/main
COPY --from=builder $APP_BUILD_DIR/templates $APP_RUN_DIR/templates

CMD ["./main"]
#ENTRYPOINT ["./main"]
