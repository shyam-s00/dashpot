.PHONY: fmt vet test test-e2e build check

fmt:            ## canonical formatting (gofmt is law)
	gofmt -w .

vet:            ## static analysis
	go vet ./...

test:           ## full suite, race detector on
	go test -race ./...

test-e2e:       ## deterministic mock-harness integration suite
	go test -race ./tests/...

build:          ## compile the dashpot binary
	go build -o bin/dashpot ./cmd/dashpot

check: fmt vet test  ## the pre-commit ritual
