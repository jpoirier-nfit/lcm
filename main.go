package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// viewMode represents different views in the TUI
type viewMode int

const (
	viewList viewMode = iota
	viewInspect
	viewLogs
	viewShell
	viewSearch
)

// Color palette and styles
var (
	// Colors
	primaryColor   = lipgloss.Color("#00D9FF")  // Cyan
	successColor   = lipgloss.Color("#00FF87")  // Green
	warningColor   = lipgloss.Color("#FFD700")  // Gold
	errorColor     = lipgloss.Color("#FF5F87")  // Pink/Red
	mutedColor     = lipgloss.Color("#626262")  // Gray
	highlightColor = lipgloss.Color("#5FD7FF")  // Light Blue

	// Title style
	titleStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Padding(0, 1)

	// Header style (for table headers)
	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#5F87AF")).
		Bold(true).
		Padding(0, 1)

	// Selected row style
	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(highlightColor).
		Bold(true)

	// Running container state
	runningStyle = lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true)

	// Exited container state
	exitedStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	// Status message style (informational)
	statusStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Padding(0, 1)

	// Warning status style (for alerts)
	warningStatusStyle = lipgloss.NewStyle().
		Foreground(warningColor).
		Bold(true).
		Padding(0, 1)

	// Filter info style
	filterStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(0, 1).
		Margin(0, 0, 1, 0)

	// Help/controls style
	helpStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Border(lipgloss.NormalBorder()).
		BorderForeground(mutedColor).
		Padding(0, 1).
		Margin(1, 0, 0, 0)

	// Key binding style
	keyStyle = lipgloss.NewStyle().
		Foreground(highlightColor).
		Bold(true)

	// Divider style
	dividerStyle = lipgloss.NewStyle().
		Foreground(mutedColor)
)

// Model represents the TUI application state
type Model struct {
	dockerClient *client.Client
	ctx          context.Context
	containers   []containerInfo
	allContainers []containerInfo // Store all containers for filtering
	cursor       int
	err          error
	loading      bool
	statusMsg    string
	currentView  viewMode
	inspectData  string
	logsData     string
	socketPath   string // Track which socket we connected to
	hideK8s      bool   // Toggle to hide k8s_ containers
	hideExited   bool   // Toggle to hide exited containers
	width        int    // Terminal width
	height       int    // Terminal height

	// Shell popup state
	shellOutput       []string // Lines of shell output
	shellInput        string   // Current input line
	shellContainerID  string   // Container ID for shell session
	shellContainerName string  // Container name for display
	shellExecID       string   // Docker exec session ID
	shellScroll       int      // Scroll position in shell output

	// Fuzzy search state
	searchInput   string         // Current search query
	searchResults []searchResult // Filtered search results
	searchCursor  int            // Selected result index
}

// searchResult represents a fuzzy search match
type searchResult struct {
	resultType  string // "container" or "command"
	display     string // Display text
	description string // Additional info
	containerID string // Container ID (for container results)
	command     string // Command key (for command results)
}

// containerInfo holds display information about a container
type containerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
	Ports  []string // Port mappings (e.g., "8080:80/tcp")
}

// containersLoadedMsg is sent when containers are loaded from Docker
type containersLoadedMsg struct {
	containers  []containerInfo
	err         error
	showRefresh bool // Whether to show "refreshed" message
}

// operationCompleteMsg is sent when a container operation completes
type operationCompleteMsg struct {
	success bool
	message string
}

// inspectDataMsg contains container inspection data
type inspectDataMsg struct {
	data string
	err  error
}

// logsDataMsg contains container logs
type logsDataMsg struct {
	data string
	err  error
}

// clearStatusMsg is sent to clear the status message
type clearStatusMsg struct{}

// tickMsg is sent periodically to trigger auto-refresh
type tickMsg time.Time

// shellReadyMsg is sent when shell exec session is ready
type shellReadyMsg struct {
	execID string
	err    error
}

// shellOutputMsg contains output from the shell
type shellOutputMsg struct {
	line string
}

// shellCommandResultMsg contains result of executing a shell command
type shellCommandResultMsg struct {
	command string
	output  string
	err     error
}

// ContainerPlatform represents a container runtime platform
type ContainerPlatform struct {
	Name       string // Display name (e.g., "Docker Desktop", "Colima")
	SocketPath string // Unix socket path or empty for DOCKER_HOST
}

// getContainerPlatforms returns all supported container platforms in priority order
func getContainerPlatforms() []ContainerPlatform {
	home := os.Getenv("HOME")
	uid := fmt.Sprintf("%d", os.Getuid())

	return []ContainerPlatform{
		// Environment variable takes highest priority
		{Name: "DOCKER_HOST", SocketPath: ""},

		// Docker Desktop
		{Name: "Docker Desktop", SocketPath: "unix:///var/run/docker.sock"},

		// Rancher Desktop
		{Name: "Rancher Desktop", SocketPath: "unix://" + home + "/.rd/docker.sock"},
		{Name: "Rancher Desktop", SocketPath: "unix://" + home + "/.docker/run/docker.sock"},

		// Colima (default profile)
		{Name: "Colima", SocketPath: "unix://" + home + "/.colima/default/docker.sock"},
		{Name: "Colima", SocketPath: "unix://" + home + "/.colima/docker.sock"},

		// Orbstack
		{Name: "Orbstack", SocketPath: "unix://" + home + "/.orbstack/run/docker.sock"},

		// Podman (macOS machine)
		{Name: "Podman", SocketPath: "unix://" + home + "/.local/share/containers/podman/machine/podman.sock"},
		{Name: "Podman", SocketPath: "unix://" + home + "/.local/share/containers/podman/machine/qemu/podman.sock"},

		// Podman (Linux user socket)
		{Name: "Podman", SocketPath: "unix:///run/user/" + uid + "/podman/podman.sock"},

		// Lima (generic)
		{Name: "Lima", SocketPath: "unix://" + home + "/.lima/default/sock/docker.sock"},
	}
}

// tryConnectDocker attempts to connect to a container runtime using multiple platforms
func tryConnectDocker(ctx context.Context) (*client.Client, string, error) {
	platforms := getContainerPlatforms()

	var lastErr error
	for _, platform := range platforms {
		var cli *client.Client
		var err error

		if platform.SocketPath == "" {
			// Try environment variables (DOCKER_HOST)
			cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		} else {
			// Try specific socket path
			cli, err = client.NewClientWithOpts(
				client.WithHost(platform.SocketPath),
				client.WithAPIVersionNegotiation(),
			)
		}

		if err != nil {
			lastErr = err
			continue
		}

		// Test the connection by pinging the daemon
		_, err = cli.Ping(ctx)
		if err != nil {
			cli.Close()
			lastErr = err
			continue
		}

		// Success! Return the platform name
		displayName := platform.Name
		if platform.SocketPath == "" {
			// Check if DOCKER_HOST is set
			dockerHost := os.Getenv("DOCKER_HOST")
			if dockerHost != "" {
				displayName = "DOCKER_HOST (" + dockerHost + ")"
			} else {
				continue // Skip if DOCKER_HOST not set
			}
		}
		return cli, displayName, nil
	}

	return nil, "", fmt.Errorf("failed to connect to container runtime: %v", lastErr)
}

func main() {
	// Initialize container client - try multiple platforms
	ctx := context.Background()
	cli, platformName, err := tryConnectDocker(ctx)
	if err != nil {
		fmt.Printf("Error: Cannot connect to any container runtime.\n")
		fmt.Printf("Tried the following platforms:\n")
		fmt.Printf("  - DOCKER_HOST environment variable\n")
		fmt.Printf("  - Docker Desktop (/var/run/docker.sock)\n")
		fmt.Printf("  - Rancher Desktop (~/.rd/docker.sock, ~/.docker/run/docker.sock)\n")
		fmt.Printf("  - Colima (~/.colima/default/docker.sock)\n")
		fmt.Printf("  - Orbstack (~/.orbstack/run/docker.sock)\n")
		fmt.Printf("  - Podman (~/.local/share/containers/podman/...)\n")
		fmt.Printf("  - Lima (~/.lima/default/sock/docker.sock)\n")
		fmt.Printf("\nError: %v\n\n", err)
		fmt.Printf("Please ensure one of the above container runtimes is running.\n")
		os.Exit(1)
	}
	defer cli.Close()

	// Initialize the Bubbletea program with alternate screen
	model := initialModel(ctx, cli)
	model.socketPath = platformName
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer (full screen)
		tea.WithMouseCellMotion(), // Enable mouse support
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

// initialModel creates the initial model for the TUI
func initialModel(ctx context.Context, cli *client.Client) Model {
	return Model{
		dockerClient: cli,
		ctx:          ctx,
		containers:   []containerInfo{},
		allContainers: []containerInfo{},
		cursor:       0,
		loading:      true,
		currentView:  viewList,
		hideK8s:      true,   // Hide k8s containers by default
		hideExited:   true,   // Hide exited containers by default
	}
}

// loadContainers fetches containers from Docker API
func (m Model) loadContainers(showRefresh bool) tea.Cmd {
	return func() tea.Msg {
		containers, err := m.dockerClient.ContainerList(m.ctx, container.ListOptions{All: true})
		if err != nil {
			return containersLoadedMsg{err: err, showRefresh: showRefresh}
		}

		var containerList []containerInfo
		for _, c := range containers {
			// Remove leading slash from container name
			name := strings.TrimPrefix(c.Names[0], "/")

		// Format ports
		var ports []string
		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				// Port is mapped to host
				ports = append(ports, fmt.Sprintf("%d:%d/%s", port.PublicPort, port.PrivatePort, port.Type))
			} else {
				// Port is exposed but not mapped
				ports = append(ports, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
			}
		}

			containerList = append(containerList, containerInfo{
				ID:     c.ID[:12], // Short ID
				Name:   name,
				Image:  c.Image,
				Status: c.Status,
				State:  c.State,
			Ports:  ports,
			})
		}

		return containersLoadedMsg{containers: containerList, showRefresh: showRefresh}
	}
}

// clearStatusAfterDelay returns a command that sends clearStatusMsg after a delay
func clearStatusAfterDelay(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// tickCmd returns a command that sends tickMsg every 1 second for auto-refresh
func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// containerCountMsg returns a properly pluralized container count message
func containerCountMsg(count int) string {
	if count == 1 {
		return "1 container"
	}
	return fmt.Sprintf("%d containers", count)
}

// filterContainers applies filters to the container list
func (m *Model) filterContainers() {
	filtered := []containerInfo{}

	for _, c := range m.allContainers {
		// Filter k8s containers
		if m.hideK8s && strings.HasPrefix(c.Name, "k8s_") {
			continue
		}

		// Filter exited containers
		if m.hideExited && c.State == "exited" {
			continue
		}

		filtered = append(filtered, c)
	}

	m.containers = filtered

	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.containers) && len(m.containers) > 0 {
		m.cursor = len(m.containers) - 1
	}
	if m.cursor < 0 && len(m.containers) > 0 {
		m.cursor = 0
	}
}

// startContainer starts the selected container
func (m Model) startContainer() tea.Msg {
	if len(m.containers) == 0 {
		return operationCompleteMsg{false, "No container selected"}
	}

	containerID := m.containers[m.cursor].ID
	err := m.dockerClient.ContainerStart(m.ctx, containerID, container.StartOptions{})
	if err != nil {
		return operationCompleteMsg{false, fmt.Sprintf("Failed to start: %v", err)}
	}

	return operationCompleteMsg{true, fmt.Sprintf("Started container %s", containerID)}
}

// stopContainer stops the selected container
func (m Model) stopContainer() tea.Msg {
	if len(m.containers) == 0 {
		return operationCompleteMsg{false, "No container selected"}
	}

	containerID := m.containers[m.cursor].ID
	timeout := 10
	err := m.dockerClient.ContainerStop(m.ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return operationCompleteMsg{false, fmt.Sprintf("Failed to stop: %v", err)}
	}

	return operationCompleteMsg{true, fmt.Sprintf("Stopped container %s", containerID)}
}

// restartContainer restarts the selected container
func (m Model) restartContainer() tea.Msg {
	if len(m.containers) == 0 {
		return operationCompleteMsg{false, "No container selected"}
	}

	containerID := m.containers[m.cursor].ID
	timeout := 10
	err := m.dockerClient.ContainerRestart(m.ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return operationCompleteMsg{false, fmt.Sprintf("Failed to restart: %v", err)}
	}

	return operationCompleteMsg{true, fmt.Sprintf("Restarted container %s", containerID)}
}
// openBrowserForContainer opens a web browser for the first available HTTP port of the selected container
func (m Model) openBrowserForContainer() tea.Cmd {
	return func() tea.Msg {
		if len(m.containers) == 0 {
			return operationCompleteMsg{false, "No container selected"}
		}

		container := m.containers[m.cursor]

		// Find the first public port
		var publicPort int
		for _, portStr := range container.Ports {
			// Parse port string like "8080:80/tcp" or "8080/tcp"
			// First try format with colon (mapped port)
			if strings.Contains(portStr, ":") {
				parts := strings.Split(portStr, ":")
				if len(parts) >= 2 {
					// Extract the host port (first part)
					portParts := strings.Split(parts[0], "/")
					if n, err := fmt.Sscanf(portParts[0], "%d", &publicPort); n == 1 && err == nil && publicPort > 0 {
						break
					}
				}
			}
		}

		if publicPort == 0 {
			if len(container.Ports) == 0 {
				return operationCompleteMsg{false, "Container has no exposed ports"}
			}
			return operationCompleteMsg{false, fmt.Sprintf("No mapped ports (have: %s)", strings.Join(container.Ports, ", "))}
		}

		url := fmt.Sprintf("http://localhost:%d", publicPort)

		// Open browser based on OS
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return operationCompleteMsg{false, "Unsupported operating system"}
		}

		if err := cmd.Start(); err != nil {
			return operationCompleteMsg{false, fmt.Sprintf("Failed to open browser: %v", err)}
		}

		return operationCompleteMsg{true, fmt.Sprintf("Opened %s in browser", url)}
	}
}


// inspectContainer retrieves detailed information about the selected container
func (m Model) inspectContainer() tea.Msg {
	if len(m.containers) == 0 {
		return inspectDataMsg{err: fmt.Errorf("no container selected")}
	}

	containerID := m.containers[m.cursor].ID
	inspect, err := m.dockerClient.ContainerInspect(m.ctx, containerID)
	if err != nil {
		return inspectDataMsg{err: err}
	}

	// Pretty print JSON
	data, err := json.MarshalIndent(inspect, "", "  ")
	if err != nil {
		return inspectDataMsg{err: err}
	}

	return inspectDataMsg{data: string(data)}
}

// viewContainerLogs retrieves logs from the selected container
func (m Model) viewContainerLogs() tea.Msg {
	if len(m.containers) == 0 {
		return logsDataMsg{err: fmt.Errorf("no container selected")}
	}

	containerID := m.containers[m.cursor].ID
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	}

	logs, err := m.dockerClient.ContainerLogs(m.ctx, containerID, options)
	if err != nil {
		return logsDataMsg{err: err}
	}
	defer logs.Close()

	// Read logs
	data, err := io.ReadAll(logs)
	if err != nil {
		return logsDataMsg{err: err}
	}

	return logsDataMsg{data: string(data)}
}

// shellIntoContainer opens an interactive shell popup for the selected container
func (m *Model) shellIntoContainer() tea.Cmd {
	if len(m.containers) == 0 {
		return nil
	}

	containerID := m.containers[m.cursor].ID
	containerName := m.containers[m.cursor].Name

	// Initialize shell state
	m.shellContainerID = containerID
	m.shellContainerName = containerName
	m.shellOutput = []string{
		fmt.Sprintf("Shell session for container: %s", containerName),
		fmt.Sprintf("Container ID: %s", containerID),
		"",
		"Type commands and press ENTER to execute.",
		"",
	}
	m.shellInput = ""
	m.shellScroll = 0

	// Switch to shell view
	m.currentView = viewShell

	return nil
}

// executeShellCommand executes a command in the container and returns the output
func (m Model) executeShellCommand(command string) tea.Cmd {
	return func() tea.Msg {
		// Create exec configuration for the command
		execConfig := container.ExecOptions{
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          []string{"/bin/sh", "-c", command},
		}

		// Create exec instance
		execResp, err := m.dockerClient.ContainerExecCreate(m.ctx, m.shellContainerID, execConfig)
		if err != nil {
			return shellCommandResultMsg{
				command: command,
				output:  "",
				err:     fmt.Errorf("failed to create exec: %w", err),
			}
		}

		// Attach to exec instance
		attachResp, err := m.dockerClient.ContainerExecAttach(m.ctx, execResp.ID, container.ExecStartOptions{})
		if err != nil {
			return shellCommandResultMsg{
				command: command,
				output:  "",
				err:     fmt.Errorf("failed to attach: %w", err),
			}
		}
		defer attachResp.Close()

		// Read all output
		output, err := io.ReadAll(attachResp.Reader)
		if err != nil {
			return shellCommandResultMsg{
				command: command,
				output:  "",
				err:     fmt.Errorf("failed to read output: %w", err),
			}
		}

		return shellCommandResultMsg{
			command: command,
			output:  string(output),
			err:     nil,
		}
	}
}

// updateSearchResults updates the search results based on current input
func (m *Model) updateSearchResults() {
	m.searchResults = []searchResult{}
	m.searchCursor = 0

	query := strings.ToLower(m.searchInput)

	// Search through containers
	for _, c := range m.containers {
		// Check if container matches the query (search name, ID, image, and ports)
		nameLower := strings.ToLower(c.Name)
		imageLower := strings.ToLower(c.Image)
		idLower := strings.ToLower(c.ID)
		portsLower := strings.ToLower(strings.Join(c.Ports, " "))

		if query == "" || strings.Contains(nameLower, query) ||
			strings.Contains(imageLower, query) || strings.Contains(idLower, query) ||
			strings.Contains(portsLower, query) {
			// Build description with ports if available
			portsStr := strings.Join(c.Ports, ", ")
			if portsStr == "" {
				portsStr = "no ports"
			}
			m.searchResults = append(m.searchResults, searchResult{
				resultType:  "container",
				display:     c.Name,
				description: fmt.Sprintf("%s | %s | %s | %s", c.ID, c.Image, portsStr, c.State),
				containerID: c.ID,
			})
		}
	}

	// Add available commands that match the query
	commands := []struct {
		key         string
		name        string
		description string
	}{
		{"s", "Start", "Start the selected container"},
		{"t", "Stop", "Stop the selected container"},
		{"R", "Restart", "Restart the selected container"},
		{"i", "Inspect", "View container details"},
		{"l", "Logs", "View container logs"},
		{"e", "Shell", "Open shell in container"},
		{"o", "Browser", "Open container port in browser"},
		{"h", "Toggle K8s", "Show/hide Kubernetes containers"},
		{"a", "Toggle Exited", "Show/hide exited containers"},
		{"r", "Refresh", "Refresh container list"},
	}

	for _, cmd := range commands {
		cmdLower := strings.ToLower(cmd.name)
		descLower := strings.ToLower(cmd.description)

		if query == "" || strings.Contains(cmdLower, query) || strings.Contains(descLower, query) {
			m.searchResults = append(m.searchResults, searchResult{
				resultType:  "command",
				display:     fmt.Sprintf("[%s] %s", cmd.key, cmd.name),
				description: cmd.description,
				command:     cmd.key,
			})
		}
	}
}

// executeSearchCommand executes a command from the search results
func (m *Model) executeSearchCommand(command string) tea.Cmd {
	switch command {
	case "s":
		m.statusMsg = "Starting container..."
		return m.startContainer
	case "t":
		m.statusMsg = "Stopping container..."
		return m.stopContainer
	case "R":
		m.statusMsg = "Restarting container..."
		return m.restartContainer
	case "i":
		m.statusMsg = "Loading inspection data..."
		return m.inspectContainer
	case "l":
		m.statusMsg = "Loading logs..."
		return m.viewContainerLogs
	case "e":
		if len(m.containers) > 0 {
			containerName := m.containers[m.cursor].Name
			m.statusMsg = fmt.Sprintf("Opening shell in %s...", containerName)
			return m.shellIntoContainer()
		}
	case "o":
		if len(m.containers) > 0 {
			m.statusMsg = "Opening browser..."
			return m.openBrowserForContainer()
		}
	case "h":
		m.hideK8s = !m.hideK8s
		m.filterContainers()
		if m.hideK8s {
			m.statusMsg = "Hiding Kubernetes containers"
		} else {
			m.statusMsg = "Showing Kubernetes containers"
		}
		return clearStatusAfterDelay(3 * time.Second)
	case "a":
		m.hideExited = !m.hideExited
		m.filterContainers()
		if m.hideExited {
			m.statusMsg = "Hiding exited containers"
		} else {
			m.statusMsg = "Showing all containers (including exited)"
		}
		return clearStatusAfterDelay(3 * time.Second)
	case "r":
		m.loading = true
		m.statusMsg = ""
		return m.loadContainers(true)
	}
	return nil
}

// Init is called when the program starts
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadContainers(false), // Don't show refresh message on initial load
		tickCmd(),               // Start auto-refresh ticker
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		// Handle different views
		switch m.currentView {
		case viewInspect, viewLogs:
			// In inspect or logs view, only allow escape to go back
			switch msg.String() {
			case "esc", "q":
				m.currentView = viewList
				m.inspectData = ""
				m.logsData = ""
			}
		case viewShell:
			// In shell view, handle shell input
			switch msg.String() {
			case "esc":
				// Exit shell view
				m.currentView = viewList
				m.shellOutput = nil
				m.shellInput = ""
				m.shellExecID = ""
			case "enter":
				// Send command to shell
				if m.shellInput != "" {
					// Show command in output
					m.shellOutput = append(m.shellOutput, "$ "+m.shellInput)
					// Execute command and get result
					cmd := m.shellInput
					m.shellInput = ""
					return m, m.executeShellCommand(cmd)
				}
			case "backspace":
				// Delete last character
				if len(m.shellInput) > 0 {
					m.shellInput = m.shellInput[:len(m.shellInput)-1]
				}
			default:
				// Add character to input
				if len(msg.String()) == 1 {
					m.shellInput += msg.String()
				}
			}
		case viewSearch:
			// In search view, handle search input
			switch msg.String() {
			case "esc":
				// Exit search view
				m.currentView = viewList
				m.searchInput = ""
				m.searchResults = nil
				m.searchCursor = 0
			case "enter":
				// Select the current result
				if len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults) {
					result := m.searchResults[m.searchCursor]
					m.currentView = viewList
					m.searchInput = ""
					m.searchResults = nil

					if result.resultType == "container" {
						// Find and select the container
						for i, c := range m.containers {
							if c.ID == result.containerID {
								m.cursor = i
								break
							}
						}
					} else if result.resultType == "command" {
						// Execute the command
						return m, m.executeSearchCommand(result.command)
					}
				}
			case "up":
				if m.searchCursor > 0 {
					m.searchCursor--
				}
			case "down":
				if m.searchCursor < len(m.searchResults)-1 {
					m.searchCursor++
				}
			case "backspace":
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
					m.updateSearchResults()
				}
			default:
				// Add character to search input
				if len(msg.String()) == 1 {
					m.searchInput += msg.String()
					m.updateSearchResults()
				}
			}
		case viewList:
			// In list view, handle all navigation and actions
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.containers)-1 {
					m.cursor++
				}
			case "r", "f5":
				// Refresh containers
				m.loading = true
				m.statusMsg = ""
				return m, m.loadContainers(true) // Show refresh message
			case "s":
				// Start container
				m.statusMsg = "Starting container..."
				return m, m.startContainer
			case "t":
				// Stop container
				m.statusMsg = "Stopping container..."
				return m, m.stopContainer
			case "R":
				// Restart container (capital R)
				m.statusMsg = "Restarting container..."
				return m, m.restartContainer
			case "i":
				// Inspect container
				m.statusMsg = "Loading inspection data..."
				return m, m.inspectContainer
			case "l":
				// View logs
				m.statusMsg = "Loading logs..."
				return m, m.viewContainerLogs
			case "e", "x":
				// Shell into container
				if len(m.containers) > 0 {
					containerName := m.containers[m.cursor].Name
					m.statusMsg = fmt.Sprintf("Opening shell in %s...", containerName)
					return m, m.shellIntoContainer()
				}
			case "o":
				// Open browser for container port
				if len(m.containers) > 0 {
					m.statusMsg = "Opening browser..."
					return m, m.openBrowserForContainer()
				}
			case "h":
				// Toggle hiding k8s containers
				m.hideK8s = !m.hideK8s
				m.filterContainers()
				if m.hideK8s {
					m.statusMsg = "Hiding Kubernetes containers"
				} else {
					m.statusMsg = "Showing Kubernetes containers"
				}
				// Clear status after 3 seconds
				return m, clearStatusAfterDelay(3 * time.Second)
			case "a":
				// Toggle showing all containers (including exited)
				m.hideExited = !m.hideExited
				m.filterContainers()
				if m.hideExited {
					m.statusMsg = "Hiding exited containers"
				} else {
					m.statusMsg = "Showing all containers (including exited)"
				}
				// Clear status after 3 seconds
				return m, clearStatusAfterDelay(3 * time.Second)
			case "/":
				// Open fuzzy search
				m.currentView = viewSearch
				m.searchInput = ""
				m.searchCursor = 0
				m.updateSearchResults() // Initialize with all results
			}
		}
	case containersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.allContainers = msg.containers
			m.filterContainers()
			if msg.showRefresh {
				m.statusMsg = "Containers refreshed"
				// Clear status after 2 seconds
				return m, clearStatusAfterDelay(2 * time.Second)
			} else if m.statusMsg == "" {
				// Only set container count if no status message exists (initial load)
				m.statusMsg = containerCountMsg(len(m.containers))
			}
			// Otherwise preserve existing status message (for background refresh)
		}
	case operationCompleteMsg:
		m.statusMsg = msg.message
		if msg.success {
			// Refresh container list after successful operation (don't show refresh message)
			m.loading = true
			return m, m.loadContainers(false)
		}
	case clearStatusMsg:
		// Clear status message and show standard status
		m.statusMsg = containerCountMsg(len(m.containers))
	case tickMsg:
		// Auto-refresh containers in background (no loading spinner, no refresh message)
		return m, tea.Batch(
			m.loadContainers(false), // Silent refresh
			tickCmd(),               // Schedule next tick
		)
	case inspectDataMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.inspectData = msg.data
			m.currentView = viewInspect
			m.statusMsg = ""
		}
	case logsDataMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.logsData = msg.data
			m.currentView = viewLogs
			m.statusMsg = ""
		}
	case shellReadyMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to create shell: %v", msg.err)
			m.currentView = viewList
		} else {
			m.shellExecID = msg.execID
			m.shellOutput = append(m.shellOutput, "Shell ready! Type commands below.", "")
		}
	case shellCommandResultMsg:
		// Display command output
		if msg.err != nil {
			m.shellOutput = append(m.shellOutput, fmt.Sprintf("Error: %v", msg.err))
		} else if msg.output != "" {
			// Split output into lines and append each
			lines := strings.Split(strings.TrimRight(msg.output, "\n"), "\n")
			m.shellOutput = append(m.shellOutput, lines...)
		}
		// Add blank line for readability
		m.shellOutput = append(m.shellOutput, "")
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.\n", m.err)
	}

	switch m.currentView {
	case viewInspect:
		return m.viewInspectMode()
	case viewLogs:
		return m.viewLogsMode()
	case viewShell:
		return m.viewShellMode()
	case viewSearch:
		return m.viewSearchMode()
	default:
		return m.viewListMode()
	}
}

// viewListMode renders the container list view
func (m Model) viewListMode() string {
	var s strings.Builder

	// Title with styled connection info
	title := "🐳 Local Container Manager (lcm)"
	if m.socketPath != "" {
		// socketPath now contains the platform name directly
		title += " [Connected to: " + m.socketPath + "]"
	}
	s.WriteString(titleStyle.Render(title) + "\n\n")

	if m.loading {
		s.WriteString("Loading containers...\n")
	} else if len(m.containers) == 0 {
		s.WriteString("No containers found.\n")
	} else {
		// Calculate how many containers we can show
		// Account for: title(2) + header(1) + divider(1) + scroll/blank(2) + status(2) + help box(8) = 16 lines overhead
		availableHeight := m.height - 16
		if availableHeight < 3 {
			availableHeight = 3 // Minimum
		}

		// Calculate scroll window
		startIdx := 0
		endIdx := len(m.containers)

		if len(m.containers) > availableHeight {
			// Need to scroll
			// Keep cursor in the middle when possible
			half := availableHeight / 2
			startIdx = m.cursor - half
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx = startIdx + availableHeight
			if endIdx > len(m.containers) {
				endIdx = len(m.containers)
				startIdx = endIdx - availableHeight
				if startIdx < 0 {
					startIdx = 0
				}
			}
		}

		// Responsive column layout with STATUS pinned to the right
		// Fixed widths: ID=12, STATE=8 (covers "running"/"exited"/"created")
		// Variable widths: NAME, IMAGE, OPENPORTS share remaining space
		// STATUS: pinned to right edge

		const (
			idWidth    = 12
			stateWidth = 8
			colSpacing = 2  // spaces between columns
			cursorCol  = 2  // space for cursor indicator
		)

		// Calculate max content widths for variable columns
		maxNameLen := len("NAME")
		maxImageLen := len("IMAGE")
		maxPortsLen := len("OPENPORTS")
		maxStatusLen := len("STATUS")

		for i := startIdx; i < endIdx; i++ {
			c := m.containers[i]
			if len(c.Name) > maxNameLen {
				maxNameLen = len(c.Name)
			}
			if len(c.Image) > maxImageLen {
				maxImageLen = len(c.Image)
			}
			portsStr := strings.Join(c.Ports, ", ")
			if portsStr == "" {
				portsStr = "-"
			}
			if len(portsStr) > maxPortsLen {
				maxPortsLen = len(portsStr)
			}
			if len(c.Status) > maxStatusLen {
				maxStatusLen = len(c.Status)
			}
		}

		// Calculate available width for variable columns
		// Layout: [cursor] ID  NAME  IMAGE  OPENPORTS  STATE  [gap]  STATUS
		fixedWidth := cursorCol + idWidth + stateWidth + (6 * colSpacing) + maxStatusLen
		availableForVariable := m.width - fixedWidth

		// Minimum widths for variable columns
		minNameWidth := 10
		minImageWidth := 10
		minPortsWidth := 9 // "OPENPORTS"

		// Distribute available space among NAME, IMAGE, OPENPORTS
		// Give proportional space based on content, with minimums
		totalContentWidth := maxNameLen + maxImageLen + maxPortsLen
		if totalContentWidth == 0 {
			totalContentWidth = 1
		}

		var nameWidth, imageWidth, portsWidth int

		if availableForVariable >= totalContentWidth {
			// Enough space for all content
			nameWidth = maxNameLen
			imageWidth = maxImageLen
			portsWidth = maxPortsLen
		} else if availableForVariable >= minNameWidth+minImageWidth+minPortsWidth {
			// Constrained: distribute proportionally with minimums
			nameWidth = max(minNameWidth, availableForVariable*maxNameLen/totalContentWidth)
			imageWidth = max(minImageWidth, availableForVariable*maxImageLen/totalContentWidth)
			portsWidth = max(minPortsWidth, availableForVariable*maxPortsLen/totalContentWidth)

			// Ensure we don't exceed available space
			for nameWidth+imageWidth+portsWidth > availableForVariable {
				if imageWidth > minImageWidth {
					imageWidth--
				} else if nameWidth > minNameWidth {
					nameWidth--
				} else if portsWidth > minPortsWidth {
					portsWidth--
				} else {
					break
				}
			}
		} else {
			// Very narrow terminal: use minimums
			nameWidth = minNameWidth
			imageWidth = minImageWidth
			portsWidth = minPortsWidth
		}

		// Header - build left part, then pin STATE and STATUS to right
		leftHeader := fmt.Sprintf(" %-*s  %-*s  %-*s  %-*s",
			idWidth, "ID", nameWidth, "NAME", imageWidth, "IMAGE", portsWidth, "OPENPORTS")

		// STATUS column width (add padding for readability)
		statusWidth := maxStatusLen + 2
		if statusWidth < 8 {
			statusWidth = 8
		}

		// Right section: STATE + STATUS pinned to right edge
		rightWidth := stateWidth + 2 + statusWidth // STATE + spacing + STATUS

		// Calculate gap to pin right section to right edge
		headerGap := m.width - len(leftHeader) - rightWidth - cursorCol
		if headerGap < 2 {
			headerGap = 2
		}

		// Build header with STATE and STATUS pinned right, both left-justified in their columns
		rightHeader := fmt.Sprintf("%-*s  %-*s", stateWidth, "STATE", statusWidth, "STATUS")
		headerText := leftHeader + strings.Repeat(" ", headerGap) + rightHeader
		// Ensure header fills full width
		if len(headerText) < m.width {
			headerText += strings.Repeat(" ", m.width-len(headerText))
		}
		s.WriteString(headerStyle.Render(headerText) + "\n")

		// Full width divider
		s.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")

		// Container list (scrollable window)
		for i := startIdx; i < endIdx; i++ {
			c := m.containers[i]

			// Truncate long names and images based on calculated widths
			name := c.Name
			if len(name) > nameWidth {
				name = name[:nameWidth-3] + "..."
			}
			image := c.Image
			if len(image) > imageWidth {
				image = image[:imageWidth-3] + "..."
			}

			// Truncate ports if needed
			portsStr := strings.Join(c.Ports, ", ")
			if portsStr == "" {
				portsStr = "-"
			}
			if len(portsStr) > portsWidth {
				portsStr = portsStr[:portsWidth-3] + "..."
			}

			// Style state based on container status
			stateText := c.State
			stateRaw := c.State // Keep unstyled version for width calculation
			if c.State == "running" {
				stateText = runningStyle.Render(c.State)
			} else if c.State == "exited" {
				stateText = exitedStyle.Render(c.State)
			}

			// Build left part of line (without cursor)
			leftPart := fmt.Sprintf(" %-*s  %-*s  %-*s  %-*s",
				idWidth, c.ID, nameWidth, name, imageWidth, image, portsWidth, portsStr)

			// Calculate gap to pin STATE and STATUS to right
			lineGap := m.width - len(leftPart) - rightWidth - cursorCol
			if lineGap < 2 {
				lineGap = 2
			}

			// Build right section with styled STATE and STATUS, both left-justified
			statePadding := strings.Repeat(" ", stateWidth-len(stateRaw))
			rightPart := stateText + statePadding + "  " + fmt.Sprintf("%-*s", statusWidth, c.Status)

			line := leftPart + strings.Repeat(" ", lineGap) + rightPart

			if i == m.cursor {
				// Highlight selected line - full width
				s.WriteString(selectedStyle.Render("▶ "+line) + "\n")
			} else {
				s.WriteString("  " + line + "\n")
			}
		}

		// Show scroll indicator if needed
		if len(m.containers) > availableHeight {
			s.WriteString(fmt.Sprintf("\nShowing %d-%d of %d containers (scroll with ↑/↓)\n",
				startIdx+1, endIdx, len(m.containers)))
		} else {
			s.WriteString("\n")
		}
	}

	// Status message - styled
	if m.statusMsg != "" {
		s.WriteString(statusStyle.Render("● "+m.statusMsg) + "\n\n")
	}

	// Help text - styled box with highlighted keys
	helpText := "Controls:\n"
	helpText += fmt.Sprintf("  Navigation: %s Up  %s Down  %s Search\n",
		keyStyle.Render("↑/k:"), keyStyle.Render("↓/j:"), keyStyle.Render("/:"))
	helpText += fmt.Sprintf("  Actions:    %s Start  %s Stop  %s Restart  %s Shell  %s Browser\n",
		keyStyle.Render("s:"), keyStyle.Render("t:"), keyStyle.Render("R:"), keyStyle.Render("e/x:"), keyStyle.Render("o:"))
	helpText += fmt.Sprintf("  Info:       %s Inspect  %s Logs\n",
		keyStyle.Render("i:"), keyStyle.Render("l:"))
	helpText += fmt.Sprintf("  Filters:    %s K8s  %s Exited\n",
		keyStyle.Render("h:"), keyStyle.Render("a:"))
	helpText += fmt.Sprintf("  Other:      %s Refresh  %s Quit",
		keyStyle.Render("r:"), keyStyle.Render("q:"))

	s.WriteString(helpStyle.Render(helpText))

	return s.String()
}

// viewInspectMode renders the container inspection view
func (m Model) viewInspectMode() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("🔍 Container Inspection") + "\n")
	// Full width divider
	dividerWidth := m.width
	if dividerWidth < 40 {
		dividerWidth = 40
	}
	s.WriteString(dividerStyle.Render(strings.Repeat("─", dividerWidth)) + "\n\n")

	if m.inspectData != "" {
		// Truncate lines that are too long for the terminal
		lines := strings.Split(m.inspectData, "\n")
		maxLines := m.height - 6 // Leave room for header and footer
		if maxLines < 10 {
			maxLines = 10
		}

		displayLines := lines
		if len(lines) > maxLines {
			displayLines = lines[:maxLines]
			s.WriteString(strings.Join(displayLines, "\n"))
			s.WriteString(fmt.Sprintf("\n\n... (showing %d of %d lines, scroll down for more)", maxLines, len(lines)))
		} else {
			s.WriteString(strings.Join(displayLines, "\n"))
		}
		s.WriteString("\n\n")
	}

	footerText := fmt.Sprintf("Press %s or %s to return to list",
		keyStyle.Render("ESC"), keyStyle.Render("q"))
	s.WriteString("\n" + helpStyle.Render(footerText) + "\n")
	return s.String()
}

// viewLogsMode renders the container logs view
func (m Model) viewLogsMode() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("📋 Container Logs (last 100 lines)") + "\n")
	// Full width divider
	dividerWidth := m.width
	if dividerWidth < 40 {
		dividerWidth = 40
	}
	s.WriteString(dividerStyle.Render(strings.Repeat("─", dividerWidth)) + "\n\n")

	if m.logsData != "" {
		// Truncate lines that are too long for the terminal
		lines := strings.Split(m.logsData, "\n")
		maxLines := m.height - 6 // Leave room for header and footer
		if maxLines < 10 {
			maxLines = 10
		}

		displayLines := lines
		if len(lines) > maxLines {
			// Show the last N lines (most recent logs)
			displayLines = lines[len(lines)-maxLines:]
			mutedStyle := lipgloss.NewStyle().Foreground(mutedColor)
			s.WriteString(mutedStyle.Render(fmt.Sprintf("... (showing last %d of %d lines)\n\n", maxLines, len(lines))))
			s.WriteString(strings.Join(displayLines, "\n"))
		} else {
			s.WriteString(strings.Join(displayLines, "\n"))
		}
		s.WriteString("\n\n")
	}

	footerText := fmt.Sprintf("Press %s or %s to return to list",
		keyStyle.Render("ESC"), keyStyle.Render("q"))
	s.WriteString("\n" + helpStyle.Render(footerText) + "\n")
	return s.String()
}

// viewShellMode renders the shell popup overlay
func (m Model) viewShellMode() string {
	// Calculate popup dimensions (80% of terminal size, min 60x20)
	popupWidth := int(float64(m.width) * 0.8)
	if popupWidth < 60 {
		popupWidth = 60
	}
	if popupWidth > m.width-4 {
		popupWidth = m.width - 4
	}

	popupHeight := int(float64(m.height) * 0.8)
	if popupHeight < 20 {
		popupHeight = 20
	}
	if popupHeight > m.height-4 {
		popupHeight = m.height - 4
	}

	// Create shell content
	var shellContent strings.Builder

	// Title
	shellTitle := fmt.Sprintf("🐚 Shell: %s", m.shellContainerName)
	shellContent.WriteString(titleStyle.Render(shellTitle) + "\n")
	shellContent.WriteString(dividerStyle.Render(strings.Repeat("─", popupWidth-4)) + "\n\n")

	// Output area (scrollable)
	outputHeight := popupHeight - 8 // Leave room for header, input, footer
	if outputHeight < 5 {
		outputHeight = 5
	}

	// Show recent output lines
	startLine := 0
	if len(m.shellOutput) > outputHeight {
		startLine = len(m.shellOutput) - outputHeight
	}

	for i := startLine; i < len(m.shellOutput); i++ {
		line := m.shellOutput[i]
		if len(line) > popupWidth-6 {
			line = line[:popupWidth-9] + "..."
		}
		shellContent.WriteString(line + "\n")
	}

	// Fill remaining space
	for i := len(m.shellOutput) - startLine; i < outputHeight; i++ {
		shellContent.WriteString("\n")
	}

	// Input line
	shellContent.WriteString("\n")
	shellContent.WriteString(dividerStyle.Render(strings.Repeat("─", popupWidth-4)) + "\n")
	inputPrompt := runningStyle.Render("$ ") + m.shellInput + "█"
	shellContent.WriteString(inputPrompt + "\n")

	// Help text
	helpText := fmt.Sprintf("%s exit shell  |  %s send command",
		keyStyle.Render("ESC"), keyStyle.Render("ENTER"))
	mutedStyle := lipgloss.NewStyle().Foreground(mutedColor)
	shellContent.WriteString(mutedStyle.Render(helpText))

	// Create popup box with border
	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 2).
		Width(popupWidth).
		Height(popupHeight)

	popup := popupStyle.Render(shellContent.String())

	// Center the popup on the screen
	vOffset := (m.height - popupHeight) / 2
	if vOffset < 0 {
		vOffset = 0
	}

	var result strings.Builder
	for i := 0; i < vOffset; i++ {
		result.WriteString("\n")
	}

	// Add horizontal centering
	hOffset := (m.width - popupWidth) / 2
	if hOffset < 0 {
		hOffset = 0
	}

	// Split popup into lines and add horizontal offset
	popupLines := strings.Split(popup, "\n")
	for _, line := range popupLines {
		result.WriteString(strings.Repeat(" ", hOffset) + line + "\n")
	}

	return result.String()
}

// viewSearchMode renders the fuzzy search overlay
func (m Model) viewSearchMode() string {
	// Calculate popup dimensions (70% of terminal size, min 50x15)
	popupWidth := int(float64(m.width) * 0.7)
	if popupWidth < 50 {
		popupWidth = 50
	}
	if popupWidth > m.width-4 {
		popupWidth = m.width - 4
	}

	popupHeight := int(float64(m.height) * 0.6)
	if popupHeight < 15 {
		popupHeight = 15
	}
	if popupHeight > m.height-4 {
		popupHeight = m.height - 4
	}

	// Create search content
	var searchContent strings.Builder

	// Title
	searchTitle := "🔍 Search"
	searchContent.WriteString(titleStyle.Render(searchTitle) + "\n")
	searchContent.WriteString(dividerStyle.Render(strings.Repeat("─", popupWidth-6)) + "\n\n")

	// Search input
	searchPrompt := primaryColor
	inputStyle := lipgloss.NewStyle().Foreground(searchPrompt).Bold(true)
	searchContent.WriteString(inputStyle.Render("> ") + m.searchInput + "█\n\n")

	// Results area
	resultHeight := popupHeight - 10
	if resultHeight < 5 {
		resultHeight = 5
	}

	if len(m.searchResults) == 0 {
		mutedStyle := lipgloss.NewStyle().Foreground(mutedColor)
		searchContent.WriteString(mutedStyle.Render("  No results found\n"))
	} else {
		// Show results with scroll if needed
		startIdx := 0
		if m.searchCursor >= resultHeight {
			startIdx = m.searchCursor - resultHeight + 1
		}
		endIdx := startIdx + resultHeight
		if endIdx > len(m.searchResults) {
			endIdx = len(m.searchResults)
		}

		for i := startIdx; i < endIdx; i++ {
			result := m.searchResults[i]

			// Format result line
			var line string
			if result.resultType == "container" {
				// Container result with icon
				line = fmt.Sprintf("📦 %s", result.display)
			} else {
				// Command result with icon
				line = fmt.Sprintf("⚡ %s", result.display)
			}

			// Truncate if too long
			maxLen := popupWidth - 10
			if len(line) > maxLen {
				line = line[:maxLen-3] + "..."
			}

			if i == m.searchCursor {
				// Highlight selected result
				searchContent.WriteString(selectedStyle.Render("▶ "+line) + "\n")
				// Show description for selected item
				descStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
				desc := result.description
				if len(desc) > maxLen {
					desc = desc[:maxLen-3] + "..."
				}
				searchContent.WriteString(descStyle.Render("    "+desc) + "\n")
			} else {
				searchContent.WriteString("  " + line + "\n")
			}
		}

		// Show scroll indicator if needed
		if len(m.searchResults) > resultHeight {
			mutedStyle := lipgloss.NewStyle().Foreground(mutedColor)
			searchContent.WriteString(mutedStyle.Render(fmt.Sprintf("\n  Showing %d-%d of %d results",
				startIdx+1, endIdx, len(m.searchResults))))
		}
	}

	// Help text at bottom
	searchContent.WriteString("\n")
	searchContent.WriteString(dividerStyle.Render(strings.Repeat("─", popupWidth-6)) + "\n")
	helpText := fmt.Sprintf("%s navigate  %s select  %s cancel",
		keyStyle.Render("↑↓"), keyStyle.Render("ENTER"), keyStyle.Render("ESC"))
	mutedStyle := lipgloss.NewStyle().Foreground(mutedColor)
	searchContent.WriteString(mutedStyle.Render(helpText))

	// Create popup box with border
	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Width(popupWidth).
		Height(popupHeight)

	popup := popupStyle.Render(searchContent.String())

	// Center the popup on the screen
	vOffset := (m.height - popupHeight) / 2
	if vOffset < 0 {
		vOffset = 0
	}

	var result strings.Builder
	for i := 0; i < vOffset; i++ {
		result.WriteString("\n")
	}

	// Add horizontal centering
	hOffset := (m.width - popupWidth) / 2
	if hOffset < 0 {
		hOffset = 0
	}

	// Split popup into lines and add horizontal offset
	popupLines := strings.Split(popup, "\n")
	for _, line := range popupLines {
		result.WriteString(strings.Repeat(" ", hOffset) + line + "\n")
	}

	return result.String()
}
