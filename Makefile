BIN := bin/ds-cli

.PHONY: build test fmt tidy all

build:
	go build -o $(BIN) .

test:
	go test ./...

fmt:
	gofmt -w main.go cmd internal

tidy:
	go mod tidy

all: fmt tidy test build
