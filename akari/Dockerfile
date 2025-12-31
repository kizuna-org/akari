FROM golang:1.25.5-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make

COPY akari/go.mod akari/go.sum ./
RUN go mod download

COPY akari/ .

RUN make generate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/akari ./cmd/akari

FROM alpine:3.23@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bin/akari /app/akari

ENV TZ=UTC

RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    chown -R akari:akari /app

USER akari

ENTRYPOINT ["/app/akari"]
