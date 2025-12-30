.PHONY: build test clean proto lint run

# Binary name
BINARY_NAME=hierachain-engine

# Go flags
GOFLAGS=-ldflags="-s -w"

# Default target
all: build

# Build the binary
build:
	go build $(GOFLAGS) -o $(BINARY_NAME).exe .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME).exe
	rm -f coverage.out coverage.html

# Run linter
lint:
	go vet ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Run the application
run: build
	./$(BINARY_NAME).exe

# Tidy dependencies
tidy:
	go mod tidy

# Download dependencies
deps:
	go mod download

# Benchmark tests
bench:
	go test -bench=. -benchmem ./...
