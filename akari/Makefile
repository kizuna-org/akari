APP_NAME := akari
BIN := bin/$(APP_NAME)
GO := go
ENV := local
GO_TOOLCHAIN := $(shell env -u GOROOT $(GO) env GOROOT)/bin/go
GOLANGCI_LINT := env -u GOROOT GOTOOLCHAIN=local $(GO_TOOLCHAIN) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2

.PHONY: init build ko-build run test fmt vet tidy generate lint clean up down down-volumes migrate-new migrate-apply migrate-status migrate-hash migrate-validate migrate-dry-run

init:
	$(GO) mod download
	if [ ! -f .env ]; then \
		cp .env.example .env; \
	fi
	if [ ! -f .env.test ]; then \
		cp .env.test.example .env.test; \
	fi

build: generate clean
	$(GO) build -o $(BIN) ./cmd/akari

ko-build:
	ko build ./cmd/akari

run: generate migrate-apply
	$(GO) run ./cmd/akari

test: generate
	$(MAKE) migrate-apply ENV=test
	ENV=test $(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

generate: init
	$(GO) generate ./...

lint: fmt vet
	$(GOLANGCI_LINT) run --fix

clean:
	rm -rf bin

up: init
	docker compose up -d

down:
	docker compose down

down-volumes:
	docker compose down -v

migrate-new: migrate-hash
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-new NAME=migration_name"; \
		exit 1; \
	fi
	atlas migrate diff $(NAME) --env=$(ENV)

migrate-apply: up
	atlas migrate apply --env=$(ENV)

migrate-status: up
	atlas migrate status --env=$(ENV)

migrate-hash: up
	atlas migrate hash --env=$(ENV)

migrate-validate: up
	atlas migrate validate --env=$(ENV)

migrate-dry-run:
	@docker compose up -d >/dev/null 2>&1
	@atlas migrate apply --dry-run --env=$(ENV)
