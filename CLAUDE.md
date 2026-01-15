# LCM Project

## Codebase Overview

LCM (Local Container Manager) is a Terminal User Interface (TUI) application for managing Docker containers across multiple container runtimes. Built with Go's Bubbletea framework, it provides an interactive interface for viewing, controlling, and debugging containers with auto-detection for Docker Desktop, Rancher Desktop, Colima, Orbstack, Podman, and Lima.

**Stack**: Go 1.25.5, Bubbletea (TUI), Docker SDK, Lipgloss (styling)

**Structure**: Single-file architecture (main.go) with message-driven TUI, multi-platform container runtime detection, and responsive terminal layout.

**Key Features**:
- Real-time container monitoring with auto-refresh
- Multi-platform auto-detection (7 container runtimes)
- Interactive shell execution
- Fuzzy search across containers
- Smart filtering (K8s, exited containers)
- Browser port launch
- Container operations (start/stop/restart/inspect/logs)

For detailed architecture, data flows, and navigation guide, see [docs/CODEBASE_MAP.md](docs/CODEBASE_MAP.md).

## Development Standards

- **Testing**: Unit tests in main_test.go covering platform detection and data structures
- **Build**: Makefile with standard targets (build, test, install, lint)
- **Code Style**: Single-file Go application following Bubbletea patterns
- **Dependencies**: Minimal external deps (Bubbletea, Lipgloss, Docker SDK)
- **Error Handling**: Graceful degradation with status messages in TUI

## Quick Start

```bash
# Build and run
make build && ./lcm

# Run tests
make test

# Install
make install
```
