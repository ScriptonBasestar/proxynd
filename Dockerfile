ARG GO_VERSION=1.22
ARG APP_BUILD_DIR=/go/src/app
ARG APP_RUN_DIR=/go/src/app


FROM golang:${GO_VERSION} as builder

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

COPY . .
RUN go get -d -v ./...
RUN go mod download
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

RUN echo $APP_BUILD_DIR $APP_RUN_DIR
WORKDIR $APP_RUN_DIR
COPY --from=builder $APP_BUILD_DIR/main .
CMD ["./main"]
