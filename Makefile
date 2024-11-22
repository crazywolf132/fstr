.PHONY: test lint bench coverage clean release

# Default target
all: test lint

# Run tests
test:
	go test -v -race ./...

# Run linter
lint:
	golangci-lint run

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean

# Create a new release
release:
	goreleaser release --clean

# Install development dependencies
setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest 