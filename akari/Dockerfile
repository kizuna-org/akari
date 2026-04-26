FROM golang:1.26.0-alpine@sha256:d4c4845f5d60c6a974c6000ce58ae079328d03ab7f721a0734277e69905473e5 AS builder

WORKDIR /build

# renovate: datasource=repology packagePrefix=alpine_3_23 versioning=loose
RUN apk add --no-cache git=2.52.0-r0 make=4.4.1-r3

COPY akari/go.mod akari/go.sum ./
RUN go mod download

COPY akari/ .
COPY kiseki/openapi/openapi.yaml ../kiseki/openapi/openapi.yaml

RUN make generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/akari ./cmd/akari

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

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
