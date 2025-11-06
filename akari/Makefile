APP_NAME := akari
BIN := bin/$(APP_NAME)
GO := go

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
	atlas migrate diff $(NAME) \
		--dir "file://internal/database/migrations" \
		--to "ent://ent/schema" \
		--dev-url "docker://postgres/17/dev?search_path=public"

migrate-apply:
	atlas migrate apply \
		--dir "file://internal/database/migrations" \
		--url "$${DATABASE_URL}"

migrate-status:
	atlas migrate status \
		--dir "file://internal/database/migrations" \
		--url "$${DATABASE_URL}"

migrate-hash:
	atlas migrate hash \
		--dir "file://internal/database/migrations"

migrate-validate:
	atlas migrate validate \
		--dir "file://internal/database/migrations"
