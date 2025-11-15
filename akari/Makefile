APP_NAME := akari
BIN := bin/$(APP_NAME)
GO := go
ENV := local

.PHONY: init build run test fmt vet tidy generate lint clean up down down-volumes migrate-new migrate-apply migrate-status migrate-hash migrate-validate migrate-dry-run

init:
	go mod download
	if [ ! -f .env ]; then \
		cp .env.example .env; \
	fi
	if [ ! -f .env.test ]; then \
		cp .env.test.example .env.test; \
	fi

build: generate clean
	$(GO) build -o $(BIN) ./cmd/akari

run: generate migrate-apply
	$(GO) run ./cmd/akari

test: generate migrate-apply
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

generate: init
	go generate ./...

lint: fmt vet
	@if [ ! -f custom-gcl ]; then \
		golangci-lint custom; \
	fi
	./custom-gcl run --fix

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
