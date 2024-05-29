# Build image
FROM golang:1.22.3-bullseye AS builder
ENV APP_HOME /build
ENV APP_USER axway

RUN mkdir -p $APP_HOME
WORKDIR $APP_HOME

# Copy necessary files
COPY . .

RUN rm -rf bin

RUN make download
RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64  make build-traceability

# Create non-root user
RUN addgroup $APP_USER && adduser --system $APP_USER --ingroup $APP_USER

RUN mkdir /data /logs && \
  apk add ca-certificates && apk update && update-ca-certificates \
  apk --no-cache add curl=7.69.1-r0 && \
  chown -R $APP_USER /data && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

RUN chgrp -R 0 /data /logs && chmod -R g=u /data && chown -R $APP_USER /data

# Base image
FROM scratch
ENV APP_HOME /build
ENV APP_USER axway

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder $APP_HOME/build/mulesoft_traceability_agent.yml /mulesoft_traceability_agent.yml
COPY --from=builder $APP_HOME/bin/traceability /traceability
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER $APP_USER
VOLUME ["/tmp", "/logs"]
HEALTHCHECK --retries=1 CMD /traceability --status || exit 1
ENTRYPOINT ["/traceability"]
