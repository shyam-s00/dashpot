.PHONY: fmt vet test test-e2e build check bench-build bench-detect bench-live

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

bench-build:    ## build dashpot, the scripted server and the runner
	go build -o bin/dashpot ./cmd/dashpot
	go build -o bin/benchsrv ./benchmark/benchsrv
	go build -o bin/benchrun ./benchmark/runner

bench-detect: bench-build  ## zero-token detection matrix
	bin/benchrun detect

bench-live: bench-build    ## live Claude runs (spends tokens)
	bin/benchrun live
