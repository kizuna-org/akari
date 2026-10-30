FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

WORKDIR /build
COPY akari/go.mod akari/go.sum ./
COPY akari/cmd ./cmd
COPY akari/internal ./internal
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /akari ./cmd/akari

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

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
