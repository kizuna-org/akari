FROM golang:1.26.2-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

WORKDIR /build
COPY akari/go.mod ./
COPY akari/cmd ./cmd
COPY akari/internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /akari ./cmd/akari

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

WORKDIR /app
RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    mkdir /data && chown akari:akari /data
COPY --from=builder /akari /app/akari
ENV AKARI_ADDR=:8080
USER akari
EXPOSE 8080
ENTRYPOINT ["/app/akari"]
