# Build image
FROM golang:1.17.8 as builder
ENV APP_HOME /build
ENV APP_USER axway

RUN mkdir -p $APP_HOME
WORKDIR $APP_HOME

# Copy necessary files
COPY . .

RUN make download
#RUN make verify
RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64  make build-discovery

# Create non-root user
RUN addgroup $APP_USER && adduser --system $APP_USER --ingroup $APP_USER
RUN chown -R $APP_USER:$APP_USER $APP_HOME

USER $APP_USER

# Base image
FROM scratch
ENV APP_HOME /build
ENV APP_USER axway
# Copy binary, user, config file and certs from previous build step
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder $APP_HOME/build/mulesoft_discovery_agent.yml /mulesoft_discovery_agent.yml
COPY --from=builder $APP_HOME/bin/discovery /discovery
COPY --from=builder /etc/passwd /etc/passwd

USER $APP_USER
VOLUME ["/tmp"]
HEALTHCHECK --retries=1 CMD curl --fail http://localhost:${STATUS_PORT:-8989}/status || exit 1
ENTRYPOINT ["/discovery"]
