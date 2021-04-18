# build stage
ARG CONTAINER_ARCH="amd64"
ARG CONTAINER_OS="linux"
ARG APP_NAME="mq-to-db"
ARG HEALTH_CHECK_PATH="health"
ARG METRICS_PORT="8080"
ARG GO_VERSION="1.16"
FROM golang:${GO_VERSION} AS build-container
RUN apt install -y git gcc make
ADD . /src
RUN cd /src && GOOS=${CONTAINER_OS} GOARCH=${CONTAINER_ARCH} make clean go-test go-build

# App container
FROM ${CONTAINER_ARCH}/busybox:glibc

LABEL maintainer="Christian González Di Antonio <christiangda@mail.com>" \
      org.opencontainers.image.authors="Christian González Di Antonio <christiangda@mail.com>" \
      org.opencontainers.image.url="https://github.com/christiangda/${APP_NAME}" \
      org.opencontainers.image.documentation="https://github.com/christiangda/${APP_NAME}" \
      org.opencontainers.image.source="https://github.com/christiangda/${APP_NAME}"

EXPOSE ${METRICS_PORT}

RUN mkdir -p /home/nobody && \
    mkdir -p /etc/mq-to-db && \
    chown -R nobody.nobody /home/nobody && \
    chown -R nobody.nobody /etc/mq-to-db

ENV HOME="/home/nobody"

COPY --from=build-container /src/mq-to-db /bin/mq-to-db
COPY --from=build-container /src/config-sample.yaml /etc/mq-to-db/config-config.yaml
RUN chmod +x /bin/mq-to-db

#HEALTHCHECK CMD wget --spider -S "http://127.0.0.1:${METRICS_PORT}/${HEALTH_CHECK_PATH}" -T 60 2>&1 || exit 1

VOLUME [ "/home/nobody", "/etc/mq-to-db"]

USER nobody
WORKDIR ${HOME}

ENTRYPOINT [ "/bin/mq-to-db" ]
#CMD  [ "/bin/mq-to-db" ]
