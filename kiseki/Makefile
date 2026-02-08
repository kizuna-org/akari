GO := go

.PHONY: init fmt vet tidy generate lint clean up down down-volumes test

init:
	$(GO) mod download

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

generate: init
	$(GO) generate ./...

lint: fmt vet
	@if [ ! -f custom-gcl ]; then \
		golangci-lint custom; \
	fi
	./custom-gcl run --fix

clean:
	rm -f gen

up:
	docker compose up -d

down:
	docker compose down

down-volumes:
	docker compose down -v

test: generate up
	$(GO) test ./... -v
