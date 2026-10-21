FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder

WORKDIR /build
COPY akari/go.mod akari/go.sum ./
COPY akari/cmd ./cmd
COPY akari/internal ./internal
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /akari ./cmd/akari

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

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
