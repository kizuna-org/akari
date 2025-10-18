APP_NAME := akari
BIN := bin/$(APP_NAME)
GO := go

.PHONY: init build run test fmt vet tidy generate lint lint-fix tools clean

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
