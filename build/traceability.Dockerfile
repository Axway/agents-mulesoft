# Build image
FROM golang:1.19.2 as builder
ENV APP_HOME /build
ENV APP_USER axway

RUN mkdir -p $APP_HOME
WORKDIR $APP_HOME

# Copy necessary files
COPY . .

RUN rm -rf bin

RUN make download
RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64  make build-traceability
# RUN export time=`date +%Y%m%d%H%M%S` && \
#   export commit_id=`git rev-parse --short HEAD` && \
#   export version=`git tag -l --sort='version:refname' | grep -Eo '[0-9]{1,}\.[0-9]{1,}\.[0-9]{1,3}$' | tail -1` && \
#   export sdk_version=`go list -m github.com/Axway/agent-sdk | awk '{print $2}' | awk -F'-' '{print substr($1, 2)}'` && \
#   export GOOS=linux && \
#   export CGO_ENABLED=0 && \
#   export GOARCH=amd64 && \
#   go build -tags static_all \
#   -ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=${time}' \
#   -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=${version}' \
#   -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=${commit_id}' \
#   -X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=${sdk_version}' \
#   -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=MuleSoftTraceabilityAgent'" \
#   -a -o bin/traceability ./cmd/traceability/main.go

# Create non-root user
RUN addgroup $APP_USER && adduser --system $APP_USER --ingroup $APP_USER

RUN mkdir /data && \
  apk add ca-certificates && apk update && update-ca-certificates \
  apk --no-cache add curl=7.69.1-r0 && \
  chown -R $APP_USER /data && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

RUN chgrp -R 0 /data && chmod -R g=u /data && chown -R $APP_USER /data

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
VOLUME ["/tmp"]
HEALTHCHECK --retries=1 CMD curl --fail http://localhost:${STATUS_PORT:-8989}/status || exit 1
ENTRYPOINT ["/traceability"]
