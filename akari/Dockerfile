FROM golang:1.25.6-alpine@sha256:98e6cffc31ccc44c7c15d83df1d69891efee8115a5bb7ede2bf30a38af3e3c92 AS builder

WORKDIR /build

RUN apk add --no-cache git=2.52.0-r0 make=4.4.1-r3

COPY akari/go.mod akari/go.sum ./
RUN go mod download

COPY akari/ .
COPY kiseki/openapi/openapi.yaml ../kiseki/openapi/openapi.yaml

RUN make generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/akari ./cmd/akari

FROM alpine:3.23@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

WORKDIR /app

RUN apk add --no-cache ca-certificates=20251003-r0 tzdata=2025c-r0 procps=4.0.5-r0

COPY --from=builder /build/bin/akari /app/akari

ENV TZ=UTC

RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    chown -R akari:akari /app

USER akari

ENTRYPOINT ["/app/akari"]
