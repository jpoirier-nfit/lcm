# Local Container Manager (lcm)

A Terminal User Interface (TUI) for managing local Docker containers, built with Go, Bubbletea, and the Docker API. Supports multiple container runtimes including **Docker Desktop**, **Rancher Desktop**, **Colima**, **Orbstack**, **Podman**, and **Lima**.

<img width="1742" height="844" alt="image" src="https://github.com/user-attachments/assets/caa98357-f6bd-4658-8d99-b81181679975" />

## Installation

### Quick Install (Recommended)

Install using Go tooling (requires Go 1.21+):

```bash
go install github.com/jpoirier-nfit/lcm@latest
```

Verify installation:
```bash
lcm
```

**Note:** Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your `PATH`.

### System Installation with Make

Install from source to `/usr/local/bin`:

Clone the repository:
```bash
git clone https://github.com/jpoirier-nfit/lcm.git
```

```bash
cd lcm
```

Install to /usr/local/bin (requires sudo):
```bash
sudo make install
```

Or install to custom location (e.g., ~/.local/bin):
```bash
make install PREFIX=$HOME/.local
```

### Build from Source

Clone and build:
```bash
git clone https://github.com/jpoirier-nfit/lcm.git
```

```bash
cd lcm
```

```bash
make build
```

Run locally:
```bash
./lcm
```

### Uninstall

If installed with make:
```bash
sudo make uninstall
```

If installed with go install:
```bash
rm $(go env GOPATH)/bin/lcm
```

## Features

- View list of Docker containers with smart filtering
- **Real-time auto-refresh** (updates every second)
- **Live status bar** showing container count and operations
- **Port display and browser launch** - View exposed ports and open them in your browser with one keypress
- Start, stop, and restart containers
- **Interactive shell popup** - Execute commands directly in containers with live output
- **Fuzzy search** - Press `/` to search containers by name, ID, image, or ports
- Inspect container details (JSON format)
- View container logs (last 100 lines)
- Interactive keyboard navigation
- **Multi-platform support** - Auto-detection of Docker Desktop, Rancher Desktop, Colima, Orbstack, Podman, and Lima
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
- One of the following container runtimes running locally:

### Supported Container Runtimes

| Platform | Socket Path(s) |
|----------|---------------|
| Docker Desktop | `/var/run/docker.sock` |
| Rancher Desktop | `~/.rd/docker.sock`, `~/.docker/run/docker.sock` |
| Colima | `~/.colima/default/docker.sock` |
| Orbstack | `~/.orbstack/run/docker.sock` |
| Podman | `~/.local/share/containers/podman/machine/podman.sock` (macOS), `/run/user/<uid>/podman/podman.sock` (Linux) |
| Lima | `~/.lima/default/sock/docker.sock` |
| DOCKER_HOST | Uses `DOCKER_HOST` environment variable if set |

The application automatically detects and connects to the first available runtime.

## Usage

After installation, simply run:

```bash
lcm
```

## Keyboard Controls

### Navigation

- `↑` or `k` - Move up in container list
- `↓` or `j` - Move down in container list
- `/` - Open fuzzy search (search by name, ID, image, or ports)

### Container Actions

- `s` - Start selected container
- `t` - Stop selected container (10 second timeout)
- `R` - Restart selected container (capital R)
- `o` - Open browser for container's first exposed port (e.g., http://localhost:8080)
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

Install dependencies:
```bash
make deps
```

Run tests:
```bash
make test
```

Build:
```bash
make build
```

Clean build artifacts:
```bash
make clean
```

## Project Structure

```
lcm/
├── main.go           # Main application and TUI models
├── main_test.go      # Unit tests
├── go.mod            # Go module dependencies
├── go.sum            # Dependency checksums
├── Makefile          # Build and run commands
└── README.md         # This file
```

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Docker Go SDK](https://github.com/docker/docker) - Docker API client
