.PHONY: all build run test test-unit test-integration lint clean verify hooks

BINARY_NAME=bin/gm

all: build

build:
	@echo "Building..."
	go build -o $(BINARY_NAME) ./cmd/gm

dev:
	@echo "Running..."
	go run ./cmd/gm/main.go

run: build
	@echo "Running..."
	./$(BINARY_NAME)

test: test-unit test-integration

test-unit:
	@echo "Running Unit Tests..."
	go test -v -race ./pkg/...

test-integration:
	@echo "Running Integration Tests..."
	go test -v ./tests/...

lint:
	@echo "Linting..."
	golangci-lint run

verify: lint test
	@echo "Verification Complete."

hooks:
	@echo "Configuring git hooks..."
	git config core.hooksPath .githooks

clean:
	@echo "Cleaning..."
	go clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf .runtime/
