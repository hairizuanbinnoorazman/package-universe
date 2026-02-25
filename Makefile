.PHONY: build run test clean install-deps

BINARY_NAME=server
CONFIG_FILE=config.yaml

build:
	go build -o bin/$(BINARY_NAME) cmd/server/*.go

run: build
	./bin/$(BINARY_NAME) serve -c $(CONFIG_FILE)

test:
	go test -v -race -cover ./...

clean:
	rm -rf bin/

install-deps:
	go mod download
	go mod tidy
