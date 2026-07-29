// Package launcher starts opencode sessions from osem.
package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// SessionManager abstracts how osem hands a session to opencode.
type SessionManager interface {
	OpenSession(sessionID, shortID string) error
	ListOpencodeSessions() ([]string, error)
}

// NewManager returns a HerdrManager when running inside a herdr environment
// (HERDR_ENV=1); otherwise it returns a DirectManager that runs opencode in
// the current terminal without any terminal multiplexer.
func NewManager(prefix string) SessionManager {
	if os.Getenv("HERDR_ENV") == "1" {
		return &HerdrManager{prefix: prefix}
	}
	return &DirectManager{prefix: prefix}
}

// ── DirectManager ────────────────────────────────────────────

// DirectManager runs opencode attached to the current terminal.
type DirectManager struct {
	prefix string
}

func (m *DirectManager) OpenSession(sessionID, shortID string) error {
	cmd := exec.Command("opencode", "-s", sessionID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *DirectManager) ListOpencodeSessions() ([]string, error) {
	return nil, nil
}

// ── HerdrManager ─────────────────────────────────────────────

type HerdrManager struct {
	prefix string
}

type herdrSplitResult struct {
	Result struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	} `json:"result"`
}

func (m *HerdrManager) OpenSession(sessionID, shortID string) error {
	// Split a new pane in the current tab
	split := exec.Command("herdr", "pane", "split",
		"--current", "--direction", "right", "--no-focus")
	output, err := split.Output()
	if err != nil {
		return fmt.Errorf("herdr: failed to split pane: %w", err)
	}

	var result herdrSplitResult
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("herdr: failed to parse split response: %w", err)
	}

	paneID := result.Result.Pane.PaneID
	if paneID == "" {
		return fmt.Errorf("herdr: split succeeded but no pane_id returned")
	}

	// Run opencode in the new pane
	cmdStr := fmt.Sprintf("opencode -s %s", sessionID)
	run := exec.Command("herdr", "pane", "run", paneID, cmdStr)
	if err := run.Run(); err != nil {
		return fmt.Errorf("herdr: failed to run opencode in pane %s: %w", paneID, err)
	}

	return nil
}

func (m *HerdrManager) ListOpencodeSessions() ([]string, error) {
	// herdr pane detection is complex; skip the tmux-style active indicator for now.
	// ponytail: return empty list — the active-session indicator is cosmetic here.
	return nil, nil
}
