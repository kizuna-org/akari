APP_NAME := akari
BIN := bin/$(APP_NAME)
GO ?= go
GO_TOOLCHAIN := $(shell env -u GOROOT $(GO) env GOROOT)/bin/go
GOLANGCI_LINT := env -u GOROOT GOTOOLCHAIN=local $(GO_TOOLCHAIN) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
COMPOSE := docker compose -f ../compose.yml

.PHONY: init build run test coverage fmt vet tidy generate lint smoke clean up down docker-build docker-check

init:
	$(GO) mod download

build:
	CGO_ENABLED=0 $(GO) build -trimpath -o $(BIN) ./cmd/akari

run:
	$(GO) run ./cmd/akari

smoke: build
	bash ../scripts/smokeHost.sh ./$(BIN)

test:
	$(GO) test -race -count=1 ./...

coverage:
	$(GO) test -race -coverprofile=coverage.out -coverpkg=./internal/... ./...
	$(GO) tool cover -func=coverage.out

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

generate:
	$(GO) generate ./...

lint: fmt vet
	$(GOLANGCI_LINT) run

clean:
	rm -rf bin

docker-build:
	$(COMPOSE) build

docker-check:
	$(COMPOSE) config --quiet

up:
	$(COMPOSE) up --build -d --wait

down:
	$(COMPOSE) down
