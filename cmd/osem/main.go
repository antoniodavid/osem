package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adruban/osem/internal/bookmarks"
	"github.com/adruban/osem/internal/db"
	"github.com/adruban/osem/internal/tmux"
	"github.com/adruban/osem/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	var (
		showVersion = flag.Bool("v", false, "show version")
		listMode    = flag.Bool("l", false, "list sessions (non-interactive)")
		sessionID   = flag.String("s", "", "open session by ID")
		statsMode   = flag.Bool("stats", false, "show session statistics")
		pruneMode   = flag.Bool("prune", false, "list sessions with default titles")
		backupID    = flag.String("backup", "", "export session to JSON")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("osem %s (%s)\n", version, commit)
		os.Exit(0)
	}

	dbPath, err := db.GetDBPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dbClient, err := db.NewClient(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	bmMgr, err := bookmarks.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not initialize bookmarks: %v\n", err)
	}

	switch {
	case *listMode:
		listSessions(dbClient)
	case *statsMode:
		showStats(dbClient)
	case *pruneMode:
		pruneSessions(dbClient)
	case *backupID != "":
		backupSession(dbClient, *backupID)
	case *sessionID != "":
		openSession(dbClient, bmMgr, *sessionID)
	default:
		runTUI(dbClient, bmMgr)
	}
}

func listSessions(dbClient *db.Client) {
	sessions, err := dbClient.ListSessions(100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-30s %-50s %s\n", "ID", "Title", "Updated")
	for _, s := range sessions {
		fmt.Printf("%-30s %-50s %s\n", s.ShortID(), s.DisplayName(), s.UpdatedAt.Format("2006-01-02 15:04"))
	}
}

func showStats(dbClient *db.Client) {
	stats, err := dbClient.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting stats: %v\n", err)
		os.Exit(1)
	}

	sessions, _ := dbClient.ListSessions(1000)

	fmt.Println("\n=== OpenCode Session Stats ===")
	fmt.Printf("Total sessions: %v\n", stats["total_sessions"])
	fmt.Printf("Total messages: %v\n", stats["total_messages"])

	if len(sessions) > 0 {
		oldest := sessions[len(sessions)-1]
		newest := sessions[0]
		fmt.Printf("Oldest session: %s\n", oldest.UpdatedAt.Format("2006-01-02"))
		fmt.Printf("Newest session: %s\n", newest.UpdatedAt.Format("2006-01-02"))
	}

	// Count by directory
	dirs := make(map[string]int)
	for _, s := range sessions {
		dirs[s.Directory]++
	}
	fmt.Printf("\nDirectories (%d unique):\n", len(dirs))
	for dir, count := range dirs {
		if count > 1 {
			fmt.Printf("  %s: %d sessions\n", truncate(dir, 50), count)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func pruneSessions(dbClient *db.Client) {
	sessions, err := dbClient.ListSessions(1000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Sessions with default titles ===")
	count := 0
	for _, s := range sessions {
		if len(s.Title) > 20 && s.Title[:20] == "New session - 2026-" {
			fmt.Printf("%-30s %s\n", s.ShortID(), s.Title)
			count++
		}
	}

	if count == 0 {
		fmt.Println("None found")
	} else {
		fmt.Printf("\n%d sessions with default titles\n", count)
		fmt.Println("To delete: osem -s <id> (manually)")
	}
}

func backupSession(dbClient *db.Client, sessionID string) {
	session, err := dbClient.GetSessionByID(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session not found: %v\n", err)
		os.Exit(1)
	}

	// Create backup directory if needed
	backupDir := filepath.Join(os.Getenv("HOME"), ".config", "osem", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating backup dir: %v\n", err)
		os.Exit(1)
	}

	// Generate filename
	timestamp := time.Now().Format("20060102-150405")
	safeTitle := session.Title
	if len(safeTitle) > 30 {
		safeTitle = safeTitle[:30]
	}
	for i, c := range safeTitle {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			safeTitle = safeTitle[:i] + "_" + safeTitle[i+1:]
		}
	}

	filename := filepath.Join(backupDir, fmt.Sprintf("%s-%s.json", session.ShortID(), timestamp))

	// Export session
	data := map[string]interface{}{
		"id":         session.ID,
		"title":      session.Title,
		"directory":  session.Directory,
		"created_at": session.CreatedAt,
		"updated_at": session.UpdatedAt,
		"short_id":   session.ShortID(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling session: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Backup saved to: %s\n", filename)
}

func openSession(dbClient *db.Client, bmMgr *bookmarks.Manager, sessionID string) {
	session, err := dbClient.GetSessionByID(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session not found: %v\n", err)
		os.Exit(1)
	}

	tmuxMgr := tmux.NewManager("opencode-")

	if err := tmuxMgr.OpenSession(session.ID, session.ShortID()); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening session: %v\n", err)
		os.Exit(1)
	}
}

func runTUI(dbClient *db.Client, bmMgr *bookmarks.Manager) {
	model, err := tui.New(dbClient, bmMgr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing TUI: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(*tui.Model); ok && m.SelectedSession() != nil {
		session := m.SelectedSession()
		tmuxMgr := tmux.NewManager("opencode-")

		if err := tmuxMgr.OpenSession(session.ID, session.ShortID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening session: %v\n", err)
			os.Exit(1)
		}
	}
}
