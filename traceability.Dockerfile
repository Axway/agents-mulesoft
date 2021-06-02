# syntax=docker/dockerfile:experimental

# Build image
FROM golang:1.16.4 as builder

RUN mkdir -p /build /app /tmp

WORKDIR /build
# Copy necessary files
COPY . . 

RUN rm -rf bin
RUN --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    --mount=type=cache,id=gopkg,target=/go/pkg \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o /app/traceability /build/cmd/traceability

# Create non-root user
RUN addgroup axway && adduser --system axway --ingroup axway

RUN mkdir /app/data && \
  apk add ca-certificates && apk update && update-ca-certificates \
  apk --no-cache add curl=7.69.1-r0 && \
  chown -R axway /data && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

RUN chgrp -R 0 /app && chmod -R g=u /app && chown -R axway /app
RUN chown 0 /build/default_mulesoft_traceability_agent.yml && chmod go-w /build/default_mulesoft_traceability_agent.yml

# Base image
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app /app
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /tmp /tmp
COPY --from=builder build/default_mulesoft_traceability_agent.yml /app/mulesoft_traceability_agent.yml

USER axway



ENTRYPOINT ["/app/traceability","--path.config", "/app"]