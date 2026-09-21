FROM golang:1.26.2-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

WORKDIR /build

# renovate: datasource=repology packagePrefix=alpine_3_23 versioning=loose
RUN apk add --no-cache git=2.52.0-r0 make=4.4.1-r3

COPY akari/go.mod akari/go.sum ./
RUN go mod download

COPY akari/ .

RUN make generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/akari ./cmd/akari

FROM alpine:3.23@sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0

WORKDIR /app

# renovate: datasource=repology packagePrefix=alpine_3_23 versioning=loose
RUN apk add --no-cache ca-certificates=20260413-r0 tzdata=2026b-r0 procps=4.0.5-r0

COPY --from=builder /build/bin/akari /app/akari

ENV TZ=UTC

RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    chown -R akari:akari /app

USER akari

ENTRYPOINT ["/app/akari"]
