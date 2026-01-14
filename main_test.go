package main

import (
	"os"
	"strings"
	"testing"
)

// TestGetContainerPlatforms verifies that the platform list is correctly generated
func TestGetContainerPlatforms(t *testing.T) {
	platforms := getContainerPlatforms()

	// Should have multiple platforms
	if len(platforms) == 0 {
		t.Error("Expected at least one platform, got none")
	}

	// Verify expected platform names are present
	expectedNames := []string{
		"DOCKER_HOST",
		"Docker Desktop",
		"Rancher Desktop",
		"Colima",
		"Orbstack",
		"Podman",
		"Lima",
	}

	foundNames := make(map[string]bool)
	for _, p := range platforms {
		foundNames[p.Name] = true
	}

	for _, expected := range expectedNames {
		if !foundNames[expected] {
			t.Errorf("Expected platform %q not found in platform list", expected)
		}
	}
}

// TestContainerPlatformSocketPaths verifies socket paths are properly formatted
func TestContainerPlatformSocketPaths(t *testing.T) {
	platforms := getContainerPlatforms()

	for _, p := range platforms {
		// Skip DOCKER_HOST which has empty socket path
		if p.SocketPath == "" {
			continue
		}

		// All socket paths should start with "unix://"
		if !strings.HasPrefix(p.SocketPath, "unix://") {
			t.Errorf("Platform %q has invalid socket path %q (should start with unix://)",
				p.Name, p.SocketPath)
		}

		// Socket paths should not contain unexpanded variables
		if strings.Contains(p.SocketPath, "$HOME") || strings.Contains(p.SocketPath, "${HOME}") {
			t.Errorf("Platform %q has unexpanded HOME variable in socket path %q",
				p.Name, p.SocketPath)
		}
	}
}

// TestContainerPlatformHomePaths verifies home directory paths are expanded
func TestContainerPlatformHomePaths(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME environment variable not set")
	}

	platforms := getContainerPlatforms()

	// Check that paths that should contain home directory do
	homePathPlatforms := []string{"Rancher Desktop", "Colima", "Orbstack", "Podman", "Lima"}

	for _, name := range homePathPlatforms {
		found := false
		for _, p := range platforms {
			if p.Name == name && strings.Contains(p.SocketPath, home) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Platform %q should have expanded HOME path but doesn't", name)
		}
	}
}

// TestSearchResultTypes verifies search result type constants
func TestSearchResultTypes(t *testing.T) {
	// Create a test search result for container
	containerResult := searchResult{
		resultType:  "container",
		display:     "test-container",
		description: "Test description",
		containerID: "abc123",
	}

	if containerResult.resultType != "container" {
		t.Errorf("Expected resultType 'container', got %q", containerResult.resultType)
	}

	// Create a test search result for command
	commandResult := searchResult{
		resultType:  "command",
		display:     "[s] Start",
		description: "Start the selected container",
		command:     "s",
	}

	if commandResult.resultType != "command" {
		t.Errorf("Expected resultType 'command', got %q", commandResult.resultType)
	}
}

// TestContainerInfoFields verifies containerInfo struct fields
func TestContainerInfoFields(t *testing.T) {
	info := containerInfo{
		ID:     "abc123456789",
		Name:   "test-container",
		Image:  "nginx:latest",
		Status: "Up 2 hours",
		State:  "running",
		Ports:  []string{"8080:80/tcp", "443:443/tcp"},
	}

	if info.ID != "abc123456789" {
		t.Errorf("Expected ID 'abc123456789', got %q", info.ID)
	}

	if info.Name != "test-container" {
		t.Errorf("Expected Name 'test-container', got %q", info.Name)
	}

	if len(info.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(info.Ports))
	}
}

// TestViewModeConstants verifies view mode constants are distinct
func TestViewModeConstants(t *testing.T) {
	modes := []viewMode{viewList, viewInspect, viewLogs, viewShell, viewSearch}
	seen := make(map[viewMode]bool)

	for _, mode := range modes {
		if seen[mode] {
			t.Errorf("Duplicate view mode value: %d", mode)
		}
		seen[mode] = true
	}
}
