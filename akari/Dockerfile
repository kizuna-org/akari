FROM golang:1.27.3-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS builder

WORKDIR /build
COPY akari/go.mod akari/go.sum ./
COPY akari/cmd ./cmd
COPY akari/internal ./internal
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /akari ./cmd/akari

FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

WORKDIR /app
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    apk add --no-cache git=2.52.0-r0 && \
    addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    mkdir /data && chown akari:akari /data
COPY --from=builder /akari /app/akari
ENV AKARI_ADDR=:8080
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/app/akari"]
