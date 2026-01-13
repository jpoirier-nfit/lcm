# Local Container Manager (lcm)

A Terminal User Interface (TUI) for managing local Docker containers, built with Go, Bubbletea, and the Docker API. Supports both **Docker Desktop** and **Rancher Desktop**.

<img width="1742" height="844" alt="image" src="https://github.com/user-attachments/assets/caa98357-f6bd-4658-8d99-b81181679975" />

## Installation

### Quick Install (Recommended)

Install using Go tooling (requires Go 1.21+):

```bash
# Install from GitHub
go install github.com/jpoirier-nfit/james_scripts/lcm@latest

# Verify installation
lcm
```

**Note:** Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your `PATH`.

### System Installation with Make

Install from source to `/usr/local/bin`:

```bash
# Clone the repository
git clone https://github.com/jpoirier-nfit/james_scripts.git
cd james_scripts/lcm

# Install to /usr/local/bin (requires sudo)
sudo make install

# Or install to custom location (e.g., ~/.local/bin)
make install PREFIX=$HOME/.local
```

### Build from Source

```bash
# Clone and build
git clone https://github.com/jpoirier-nfit/james_scripts.git
cd james_scripts/lcm
make build

# Run locally
./lcm
```

### Uninstall

```bash
# If installed with make
sudo make uninstall

# If installed with go install
rm $(go env GOPATH)/bin/lcm
```

## Features

- View list of Docker containers with smart filtering
- **Real-time auto-refresh** (updates every second)
- **Live status bar** showing container count and operations
- Start, stop, and restart containers
- **Interactive shell popup** - Execute commands directly in containers with live output
- Inspect container details (JSON format)
- View container logs (last 100 lines)
- Interactive keyboard navigation
- Auto-detection of Docker Desktop or Rancher Desktop
- Real-time container state display
- **Smart filters (enabled by default):**
  - Hide Kubernetes system containers (k8s\_\*) - toggle with `h`
  - Hide exited containers - toggle with `a`
- **Full-screen terminal UI** with automatic scrolling for long lists
- **Full-width display** that adapts to any terminal size
- **Beautiful color styling** with visual separation of sections:
  - Cyan/blue header and titles
  - Green highlights for running containers
  - Gray muted exited containers
  - Highlighted selected row with blue background
  - Bordered boxes for filters and controls
  - Color-coded key bindings
- Responsive design that adapts to terminal size
- Mouse support for scrolling and navigation

## Requirements

- Go 1.21+
- Docker Desktop **OR** Rancher Desktop running locally
- The application automatically tries multiple socket paths:
  - `/var/run/docker.sock` (Docker Desktop)
  - `~/.rd/docker.sock` (Rancher Desktop - older versions)
  - `~/.docker/run/docker.sock` (Rancher Desktop - newer versions)
  - `DOCKER_HOST` environment variable

## Usage

After installation, simply run:

```bash
lcm
```

## Keyboard Controls

### Navigation

- `↑` or `k` - Move up in container list
- `↓` or `j` - Move down in container list

### Container Actions

- `s` - Start selected container
- `t` - Stop selected container (10 second timeout)
- `R` - Restart selected container (capital R)
- `e` or `x` - Open interactive shell popup for selected container
  - Opens a centered popup window with shell interface
  - Type commands and press `ENTER` to execute
  - See real-time output from the container
  - Press `ESC` to close shell and return to container list
  - Press `BACKSPACE` to delete input characters

### Information

- `i` - Inspect container (view detailed JSON)
- `l` - View container logs (last 100 lines)

### Filters (Active by Default)

- `h` - Toggle hide/show Kubernetes containers (k8s\_\*)
- `a` - Toggle hide/show exited containers (All/Active only)

### Other

- `r` or `F5` - Refresh container list
- `ESC` or `q` - Go back / Quit
- `Ctrl+C` - Force quit

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Build
make build

# Clean build artifacts
make clean
```

## Project Structure

```
lcm/
├── main.go           # Main application and TUI models
├── go.mod            # Go module dependencies
├── go.sum            # Dependency checksums
├── Makefile          # Build and run commands
└── README.md         # This file
```

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Docker Go SDK](https://github.com/docker/docker) - Docker API client
