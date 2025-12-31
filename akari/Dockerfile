FROM golang:1.25.5-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder

WORKDIR /build

RUN apk add --no-cache git make

COPY akari/go.mod akari/go.sum ./
RUN go mod download

COPY akari/ .

RUN make generate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/akari ./cmd/akari

FROM alpine:3.23@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata procps

COPY --from=builder /build/bin/akari /app/akari

ENV TZ=UTC

RUN addgroup -g 1000 akari && \
    adduser -D -u 1000 -G akari akari && \
    chown -R akari:akari /app

USER akari

ENTRYPOINT ["/app/akari"]
