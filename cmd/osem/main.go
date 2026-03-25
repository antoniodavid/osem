package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		statsMode   = flag.Bool("stats", false, "show session statistics")
		pruneMode   = flag.Bool("prune", false, "list sessions with default titles")
		pruneDelete = flag.Bool("delete", false, "delete pruned sessions (use with -prune)")
		backupID    = flag.String("backup", "", "export session to JSON")
		searchQuery = flag.String("grep", "", "search sessions by title")
		infoID      = flag.String("info", "", "show session details")
		sessionID   = flag.String("s", "", "open session by ID")
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
		pruneSessions(dbClient, *pruneDelete)
	case *backupID != "":
		backupSession(dbClient, *backupID)
	case *searchQuery != "":
		searchSessions(dbClient, *searchQuery)
	case *infoID != "":
		showSessionInfo(dbClient, *infoID)
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

	fmt.Println("\n╭─────────────────────────────────────╮")
	fmt.Println("│     OpenCode Session Statistics      │")
	fmt.Println("╰─────────────────────────────────────╯")
	fmt.Printf("\n  Total sessions:  %d\n", stats["total_sessions"])
	fmt.Printf("  Total messages:  %d\n", stats["total_messages"])

	if len(sessions) > 0 {
		oldest := sessions[len(sessions)-1]
		newest := sessions[0]
		fmt.Printf("  Date range:      %s → %s\n",
			oldest.UpdatedAt.Format("2006-01-02"),
			newest.UpdatedAt.Format("2006-01-02"))
	}

	// Count by directory (top 10)
	dirs := make(map[string]int)
	for _, s := range sessions {
		dirs[s.Directory]++
	}

	fmt.Println("\n  Top directories:")
	count := 0
	for dir, cnt := range dirs {
		if count >= 10 {
			break
		}
		fmt.Printf("    %d: %s (%d sessions)\n", count+1, truncate(dir, 45), cnt)
		count++
	}

	// Recent activity (last 7 days)
	now := time.Now()
	recentCount := 0
	for _, s := range sessions {
		if now.Sub(s.UpdatedAt) < 7*24*time.Hour {
			recentCount++
		}
	}
	fmt.Printf("\n  Recent activity (7 days): %d sessions\n", recentCount)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func pruneSessions(dbClient *db.Client, delete bool) {
	sessions, err := dbClient.ListSessions(1000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
		os.Exit(1)
	}

	var toDelete []string
	fmt.Println("=== Sessions with default titles ===")

	for _, s := range sessions {
		// Check for default title patterns
		if strings.HasPrefix(s.Title, "New session -") {
			fmt.Printf("%-30s %s\n", s.ShortID(), s.Title)
			toDelete = append(toDelete, s.ID)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("None found")
		return
	}

	fmt.Printf("\n%d sessions with default titles\n", len(toDelete))

	if delete {
		fmt.Print("\nDelete all? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)

		if strings.ToLower(confirm) == "y" {
			for _, id := range toDelete {
				if err := dbClient.DeleteSession(id); err != nil {
					fmt.Fprintf(os.Stderr, "Error deleting %s: %v\n", id[:12], err)
				} else {
					fmt.Printf("Deleted: %s\n", id[:12])
				}
			}
			fmt.Printf("\nDeleted %d sessions\n", len(toDelete))
		}
	} else {
		fmt.Println("\nTo delete: osem -prune -delete")
	}
}

func searchSessions(dbClient *db.Client, query string) {
	sessions, err := dbClient.SearchSessions(query, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
		os.Exit(1)
	}

	if len(sessions) == 0 {
		fmt.Printf("No sessions found matching: %s\n", query)
		return
	}

	fmt.Printf("Found %d sessions:\n\n", len(sessions))
	for _, s := range sessions {
		fmt.Printf("%-30s %s\n", s.ShortID(), s.Title)
	}
}

func showSessionInfo(dbClient *db.Client, sessionID string) {
	session, err := dbClient.GetSessionByID(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session not found: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n╭─────────────────────────────────────╮")
	fmt.Println("│          Session Details            │")
	fmt.Println("╰─────────────────────────────────────╯")
	fmt.Printf("\n  ID:         %s\n", session.ID)
	fmt.Printf("  Short ID:   %s\n", session.ShortID())
	fmt.Printf("  Title:      %s\n", session.Title)
	fmt.Printf("  Directory:  %s\n", session.Directory)
	fmt.Printf("  Created:    %s\n", session.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("  Updated:    %s\n", session.UpdatedAt.Format("2006-01-02 15:04"))

	// Message count
	msgCount, err := dbClient.GetMessageCount(session.ID)
	if err == nil {
		fmt.Printf("  Messages:   %d\n", msgCount)
	}

	// Check if in tmux
	fmt.Println("\n  To open:    osem -s " + session.ShortID())
}

func backupSession(dbClient *db.Client, sessionID string) {
	session, err := dbClient.GetSessionByID(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session not found: %v\n", err)
		os.Exit(1)
	}

	backupDir := filepath.Join(os.Getenv("HOME"), ".config", "osem", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating backup dir: %v\n", err)
		os.Exit(1)
	}

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

	fmt.Printf("✓ Backup saved to: %s\n", filename)
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
