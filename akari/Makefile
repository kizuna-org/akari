APP_NAME := akari
BIN := bin/$(APP_NAME)
GO := go
ENV := local

.PHONY: init build run test fmt vet tidy generate lint lint-fix tools clean migrate-new migrate-apply migrate-status migrate-validate migrate-lint

init:
	go mod download
	if [ ! -f .env ]; then \
		cp .env.example .env; \
	fi

build: init generate
	$(GO) build -o $(BIN) ./cmd/akari

run: init generate
	$(GO) run ./cmd/akari

 test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

 tidy:
	$(GO) mod tidy

generate: init
	go generate ./...

lint:
	@if [ ! -f custom-gcl ]; then \
		golangci-lint custom; \
	fi
	./custom-gcl run --fix

clean:
	rm -rf bin

migrate-new:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-new NAME=migration_name"; \
		exit 1; \
	fi
	atlas migrate diff $(NAME) --env=$(ENV)

migrate-apply:
	atlas migrate apply --env=$(ENV)

migrate-status:
	atlas migrate status --env=$(ENV)

migrate-hash:
	atlas migrate hash --env=$(ENV)

migrate-validate:
	atlas migrate validate --env=$(ENV)
