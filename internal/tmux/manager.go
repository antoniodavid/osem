package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Manager struct {
	prefix string
}

func NewManager(prefix string) *Manager {
	return &Manager{prefix: prefix}
}

func (m *Manager) sessionName(shortID string) string {
	return m.prefix + shortID
}

func (m *Manager) IsInTmux() bool {
	return os.Getenv("TMUX") != ""
}

func (m *Manager) SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

func (m *Manager) ListOpencodeSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-f", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}

	var sessions []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, m.prefix) {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

func (m *Manager) CreateSession(name, cwd string, command string) error {
	args := []string{"new-session", "-d", "-s", name, "-c", cwd}
	if command != "" {
		args = append(args, command)
	}

	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

func (m *Manager) AttachSession(name string) error {
	if m.IsInTmux() {
		cmd := exec.Command("tmux", "switch-client", "-t", name)
		return cmd.Run()
	}

	cmd := exec.Command("tmux", "attach", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

func (m *Manager) OpenSession(sessionID, shortID string) error {
	name := m.sessionName(shortID)

	// If session exists, switch to it
	if m.SessionExists(name) {
		return m.AttachSession(name)
	}

	// Build opencode command with full session ID
	// OpenCode handles directory change internally with -s flag
	cmdStr := fmt.Sprintf("opencode -s %s", sessionID)

	// If already in tmux, use current window instead of creating new session
	if m.IsInTmux() {
		// Get current session name for window naming
		currentSession, _ := m.GetActiveSession()
		windowName := fmt.Sprintf("opencode-%s", shortID)

		// Create new window with descriptive name (no -c needed, opencode handles cwd)
		windowArgs := []string{"new-window", "-n", windowName}
		newWindowCmd := exec.Command("tmux", windowArgs...)
		if err := newWindowCmd.Run(); err != nil {
			return fmt.Errorf("failed to create new window: %w", err)
		}

		// Send opencode command to the new window
		sendCmd := exec.Command("tmux", "send-keys", "-t:", cmdStr, "Enter")
		if err := sendCmd.Run(); err != nil {
			return fmt.Errorf("failed to send command: %w", err)
		}

		// Log what we did
		if currentSession != "" {
			fmt.Fprintf(os.Stderr, "Created window '%s' in session '%s'\n", windowName, currentSession)
		}
		return nil
	}

	// Not in tmux - create new session
	windowName := fmt.Sprintf("opencode-%s", shortID)
	if err := m.CreateSession(name, "", ""); err != nil {
		return err
	}

	// Rename the default window
	renameCmd := exec.Command("tmux", "rename-window", "-t", name+":0", windowName)
	renameCmd.Run()

	// Send opencode command to the session
	sendCmd := exec.Command("tmux", "send-keys", "-t", name, cmdStr, "Enter")
	if err := sendCmd.Run(); err != nil {
		return err
	}

	return m.AttachSession(name)
}

func (m *Manager) GetActiveSession() (string, error) {
	if !m.IsInTmux() {
		return "", fmt.Errorf("not in a tmux session")
	}

	session := os.Getenv("TMUX")
	if session == "" {
		return "", fmt.Errorf("could not determine current tmux session")
	}

	// Get session name
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (m *Manager) IsOpencodeSession(name string) bool {
	return strings.HasPrefix(name, m.prefix)
}
