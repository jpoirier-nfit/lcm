.PHONY: build run clean test install uninstall help fmt lint

# Installation prefix (can be overridden: make install PREFIX=/custom/path)
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

# Default target - show help
.DEFAULT_GOAL := help

# Build the application
build:
	go build -o lcm .

# Run the application
run:
	go run .

# Clean build artifacts
clean:
	rm -f lcm docker-tui

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install to system (requires sudo if installing to /usr/local)
install: build
	@echo "Installing lcm to $(BINDIR)..."
	@mkdir -p $(BINDIR)
	@install -m 755 lcm $(BINDIR)/lcm
	@echo "lcm installed successfully to $(BINDIR)/lcm"
	@echo "Run 'lcm' from anywhere to start the application"

# Uninstall from system
uninstall:
	@echo "Removing lcm from $(BINDIR)..."
	@rm -f $(BINDIR)/lcm
	@echo "lcm uninstalled successfully"

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

# Show help
help:
	@echo "Local Container Manager (lcm) - Available targets:"
	@echo ""
	@echo "  make build      - Build the application"
	@echo "  make run        - Run the application"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make deps       - Install/update dependencies"
	@echo "  make fmt        - Format code with go fmt"
	@echo "  make lint       - Run linter (requires golangci-lint)"
	@echo "  make install    - Install to $(PREFIX)/bin (may require sudo)"
	@echo "  make uninstall  - Uninstall from $(PREFIX)/bin"
	@echo ""
	@echo "  Override installation path: make install PREFIX=/custom/path"
