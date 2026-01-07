GO := go

.PHONY: init fmt vet tidy generate clean

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

clean:
	rm -f gen
