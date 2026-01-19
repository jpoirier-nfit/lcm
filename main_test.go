package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

// TestPasteInShellView verifies paste handling in shell view
func TestPasteInShellView(t *testing.T) {
	model := Model{
		currentView: viewShell,
		shellInput:  "echo ",
	}

	// Simulate paste event with KeyMsg
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("hello world"),
		Paste: true,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	expected := "echo hello world"
	if m.shellInput != expected {
		t.Errorf("Expected shell input %q, got %q", expected, m.shellInput)
	}
}

// TestPasteInSearchView verifies paste handling in search view
func TestPasteInSearchView(t *testing.T) {
	model := Model{
		currentView: viewSearch,
		searchInput: "nginx",
		containers: []containerInfo{
			{Name: "nginx-web", Image: "nginx:latest"},
			{Name: "postgres-db", Image: "postgres:14"},
		},
	}

	// Simulate paste event
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("-web"),
		Paste: true,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	expected := "nginx-web"
	if m.searchInput != expected {
		t.Errorf("Expected search input %q, got %q", expected, m.searchInput)
	}
}

// TestPasteInListView verifies paste is ignored in list view
func TestPasteInListView(t *testing.T) {
	model := Model{
		currentView: viewList,
	}

	// Simulate paste event - should be ignored in list view
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("some pasted text"),
		Paste: true,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Model should be unchanged in list view
	if m.currentView != viewList {
		t.Errorf("View should remain as viewList")
	}
}

// TestPasteEmptyString verifies handling of empty paste
func TestPasteEmptyString(t *testing.T) {
	model := Model{
		currentView: viewShell,
		shellInput:  "test",
	}

	// Simulate empty paste event
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(""),
		Paste: true,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Input should remain unchanged
	if m.shellInput != "test" {
		t.Errorf("Expected shell input 'test', got %q", m.shellInput)
	}
}

// TestPasteSpecialCharacters verifies paste with special characters
func TestPasteSpecialCharacters(t *testing.T) {
	model := Model{
		currentView: viewShell,
		shellInput:  "",
	}

	// Simulate paste with special characters
	specialText := "echo 'test' && ls -la /tmp"
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(specialText),
		Paste: true,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.shellInput != specialText {
		t.Errorf("Expected shell input %q, got %q", specialText, m.shellInput)
	}
}

// TestPasteMultiline verifies paste with newlines
func TestPasteMultiline(t *testing.T) {
	model := Model{
		currentView: viewShell,
		shellInput:  "",
	}

	// Simulate paste with newlines
	multilineText := "line1\nline2\nline3"
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(multilineText),
		Paste: true,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if m.shellInput != multilineText {
		t.Errorf("Expected shell input %q, got %q", multilineText, m.shellInput)
	}
}

// TestDestroyConfirmationTriggered verifies that 'd' key triggers confirmation dialog
func TestDestroyConfirmationTriggered(t *testing.T) {
	model := Model{
		currentView: viewList,
		containers: []containerInfo{
			{ID: "abc123", Name: "test-container"},
		},
		cursor: 0,
	}

	// Simulate 'd' key press
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("d"),
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Should be in confirmation mode
	if !m.confirmingDestroy {
		t.Error("Expected confirmingDestroy to be true")
	}

	// Should have stored the container ID
	if m.containerToDestroy != "abc123" {
		t.Errorf("Expected containerToDestroy 'abc123', got %q", m.containerToDestroy)
	}

	// Status message should ask for confirmation
	if !strings.Contains(m.statusMsg, "Destroy container") {
		t.Errorf("Expected confirmation message in statusMsg, got %q", m.statusMsg)
	}
}

// TestDestroyConfirmationCancelled verifies that 'n' cancels the destroy
func TestDestroyConfirmationCancelled(t *testing.T) {
	model := Model{
		currentView:        viewList,
		confirmingDestroy:  true,
		containerToDestroy: "abc123",
		containers: []containerInfo{
			{ID: "abc123", Name: "test-container"},
		},
	}

	// Simulate 'n' key press
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("n"),
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Should no longer be in confirmation mode
	if m.confirmingDestroy {
		t.Error("Expected confirmingDestroy to be false after cancellation")
	}

	// Container ID should be cleared
	if m.containerToDestroy != "" {
		t.Errorf("Expected containerToDestroy to be empty, got %q", m.containerToDestroy)
	}

	// Status message should indicate cancellation
	if !strings.Contains(m.statusMsg, "cancelled") {
		t.Errorf("Expected cancellation message in statusMsg, got %q", m.statusMsg)
	}
}

// TestDestroyConfirmationEscapeCancels verifies that ESC cancels the destroy
func TestDestroyConfirmationEscapeCancels(t *testing.T) {
	model := Model{
		currentView:        viewList,
		confirmingDestroy:  true,
		containerToDestroy: "abc123",
		containers: []containerInfo{
			{ID: "abc123", Name: "test-container"},
		},
	}

	// Simulate ESC key press
	msg := tea.KeyMsg{
		Type: tea.KeyEsc,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Should no longer be in confirmation mode
	if m.confirmingDestroy {
		t.Error("Expected confirmingDestroy to be false after ESC")
	}
}

// TestDestroyIgnoresOtherKeys verifies other keys are ignored during confirmation
func TestDestroyIgnoresOtherKeys(t *testing.T) {
	model := Model{
		currentView:        viewList,
		confirmingDestroy:  true,
		containerToDestroy: "abc123",
		containers: []containerInfo{
			{ID: "abc123", Name: "test-container"},
		},
	}

	// Simulate 's' key press (which normally starts a container)
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("s"),
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Should still be in confirmation mode
	if !m.confirmingDestroy {
		t.Error("Expected confirmingDestroy to remain true")
	}

	// Container ID should still be set
	if m.containerToDestroy != "abc123" {
		t.Errorf("Expected containerToDestroy 'abc123', got %q", m.containerToDestroy)
	}
}
