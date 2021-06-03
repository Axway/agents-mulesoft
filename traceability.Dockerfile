# Build image
FROM golang:1.16.4 as builder
ENV APP_HOME /build
ENV APP_USER axway

RUN mkdir -p $APP_HOME /app

WORKDIR $APP_HOME
# Copy necessary files
COPY . . 

RUN rm -rf bin

RUN make download
RUN make verify
RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64  make build-trace

# Create non-root user
RUN addgroup $APP_USER && adduser --system $APP_USER --ingroup $APP_USER

RUN mkdir /app/data && \
  apk add ca-certificates && apk update && update-ca-certificates \
  apk --no-cache add curl=7.69.1-r0 && \
  chown -R $APP_USER /data && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

RUN chgrp -R 0 /app && chmod -R g=u /app && chown -R $APP_USER /app
RUN chown 0 $APP_HOME/default_mulesoft_traceability_agent.yml && chmod go-w $APP_HOME/default_mulesoft_traceability_agent.yml

# Base image
FROM scratch
ENV APP_HOME /build
ENV APP_USER axway

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app /app
COPY --from=builder $APP_HOME/default_mulesoft_traceability_agent.yml /app/mulesoft_traceability_agent.yml
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER $APP_USER
VOLUME ["/tmp"]
HEALTHCHECK --retries=1 CMD curl --fail http://localhost:${STATUS_PORT:-8989}/status || exit 1
ENTRYPOINT ["/app/traceability","--path.config", "/app"]