FROM golang:1.13 as builder

RUN mkdir -p /build

WORKDIR /build

# Copy necessary files
ADD . .

#change the ./cmd/alstraceability
RUN rm -rf bin
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -o bin/traceability ./cmd/traceability

# Create non-root user
RUN addgroup alstraceability && adduser --system alstraceability --ingroup alstraceability
RUN chown -R alstraceability:alstraceability bin/traceability
RUN chown -R alstraceability:alstraceability default_mulesoft_traceability_agent.yml
RUN chmod go-w default_mulesoft_traceability_agent.yml
USER alstraceability


# Base image
FROM alpine:3.12.3

# Copy binary, user, config file and certs from previous build step
# Copying Public  and Private key directly into the image -> imprvement needed
COPY --from=builder /build/public_key.pem /public_key.pem
COPY --from=builder /build/private_key.pem /private_key.pem
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /build/default_mulesoft_traceability_agent.yml /mulesoft_traceability_agent.yml
COPY --from=builder /build/bin/traceability /alstraceability
COPY --from=builder /etc/passwd /etc/passwd

RUN mkdir /data && \
  apk add ca-certificates && apk update && update-ca-certificates \
  apk --no-cache add curl=7.69.1-r0 && \
  chown -R alstraceability /data && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

USER alstraceability

ENTRYPOINT ["/alstraceability"]
