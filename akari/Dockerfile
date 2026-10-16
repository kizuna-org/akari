FROM golang:1.26.2-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

WORKDIR /build
COPY akari/go.mod ./
COPY akari/cmd ./cmd
COPY akari/internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /akari ./cmd/akari

FROM alpine:3.23@sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0

WORKDIR /app
RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    mkdir /data && chown akari:akari /data
COPY --from=builder /akari /app/akari
ENV AKARI_ADDR=:8080
USER akari
EXPOSE 8080
ENTRYPOINT ["/app/akari"]
