BINARY_NAME=api

.PHONY: run build test lint

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run
