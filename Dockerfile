ARG ARCH="amd64"
ARG PROJECT_NAME="mq-to-db"
ARG HEALTH_CHECK_PATH="health"
ARG METRICS_PORT="8080"

FROM ${ARCH}/busybox:glibc

LABEL maintainer="Mantainer User <mantainer.user@mail.com>" \
      org.opencontainers.image.authors="Mantainer User <mantainer.user@mail.com>" \
      org.opencontainers.image.url="https://github.com/christiangda/${PROJECT_NAME}" \
      org.opencontainers.image.documentation="https://github.com/christiangda/${PROJECT_NAME}" \
      org.opencontainers.image.source="https://github.com/christiangda/${PROJECT_NAME}"

EXPOSE  ${METRICS_PORT}

RUN mkdir -p /home/nobody && \
    mkdir -p /etc/mq-to-db && \
    chown -R nobody.nogroup /home/nobody && \
    chown -R nobody.nogroup /etc/mq-to-db

ENV HOME="/home/nobody"

COPY mq-to-db /bin/mq-to-db
COPY config-sample.yaml /etc/mq-to-db/config.yaml
RUN chmod +x /bin/mq-to-db

#HEALTHCHECK CMD wget --spider -S "http://127.0.0.1:${METRICS_PORT}/${HEALTH_CHECK_PATH}" -T 60 2>&1 || exit 1

VOLUME [ "/home/nobody", "/etc/mq-to-db"]

USER nobody
WORKDIR ${HOME}

#ENTRYPOINT [ "/bin/mq-to-db" ]
CMD  [ "/bin/mq-to-db" ]
