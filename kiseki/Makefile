GO := go

.PHONY: init fmt vet tidy generate lint clean

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
