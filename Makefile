.PHONY: build run clean test install uninstall

# Installation prefix (can be overridden: make install PREFIX=/custom/path)
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

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
