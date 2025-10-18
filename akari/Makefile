APP_NAME := akari
BIN := bin/$(APP_NAME)
GO := go

.PHONY: init build run test fmt vet tidy generate lint lint-fix tools clean

init:
	go mod download

build:
	$(GO) build -o $(BIN) ./cmd/akari

run:
	$(GO) run ./cmd/akari

 test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

 tidy:
	$(GO) mod tidy

generate:
	go generate ./...

lint:
	@if [ ! -f custom-gcl ]; then \
		golangci-lint custom; \
	fi
	./custom-gcl run --fix

clean:
	rm -rf bin
