APP_NAME := nina
BIN := bin/$(APP_NAME)
GO := go

.PHONY: build run test fmt vet tidy lint lint-fix tools clean

build:
	$(GO) build -o $(BIN) ./cmd/nina

run:
	$(GO) run ./cmd/nina

 test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

 tidy:
	$(GO) mod tidy

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

clean:
	rm -rf bin
