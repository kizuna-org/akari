FROM golang:1.26.2-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

WORKDIR /build

# renovate: datasource=repology packagePrefix=alpine_3_23 versioning=loose
RUN apk add --no-cache git=2.52.0-r0 make=4.4.1-r3

COPY akari/go.mod akari/go.sum ./
RUN go mod download

COPY akari/ .
COPY kiseki/openapi/openapi.yaml ../kiseki/openapi/openapi.yaml

RUN make generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/akari ./cmd/akari

FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

WORKDIR /app

# renovate: datasource=repology packagePrefix=alpine_3_23 versioning=loose
RUN apk add --no-cache ca-certificates=20251003-r0 tzdata=2025c-r0 procps=4.0.5-r0

COPY --from=builder /build/bin/akari /app/akari

ENV TZ=UTC

RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    chown -R akari:akari /app

USER akari

ENTRYPOINT ["/app/akari"]
