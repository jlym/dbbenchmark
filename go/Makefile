.PHONY: build
build:
	go build -o build/server github.com/jlym/dbbenchmark/go/cmd/server

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go vet ./...

.PHONY: test
test:
	go test -v ./...
