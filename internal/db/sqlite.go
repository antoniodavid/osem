package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/adruban/osem/internal/models"
)

type Client struct {
	db *sql.DB
	// sessionsTable is the active session table: "session_v2" on OpenCode v2+
	// storage, "session" as fallback for older databases. messagesTable is the
	// matching message table ("session_message" or "message").
	sessionsTable string
	messagesTable string
}

func NewClient(dbPath string) (*Client, error) {
	expandedPath := expandHome(dbPath)
	db, err := sql.Open("sqlite", expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	sessionsTable, messagesTable, err := detectSchema(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to detect opencode schema in %s: %w", expandedPath, err)
	}
	return &Client{db: db, sessionsTable: sessionsTable, messagesTable: messagesTable}, nil
}

// detectSchema picks the session/message tables to query. OpenCode v2 moved
// live sessions to session_v2/session_message; legacy session/message tables
// are kept as a fallback for older databases.
func detectSchema(db *sql.DB) (string, string, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'session_v2'`).Scan(&count); err != nil {
		return "", "", err
	}
	if count > 0 {
		return "session_v2", "session_message", nil
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'session'`).Scan(&count); err != nil {
		return "", "", err
	}
	if count > 0 {
		return "session", "message", nil
	}
	return "", "", fmt.Errorf("no session table found (expected session_v2 or session)")
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) ListSessions(limit int) ([]models.Session, error) {
	query := fmt.Sprintf(`
		SELECT 
			s.id,
			COALESCE(s.title, 'Untitled Session') as title,
			s.directory,
			s.project_id,
			s.slug,
			s.time_updated,
			s.time_created,
			COALESCE(s.agent, '') as agent,
			COALESCE(s.model, '') as model,
			COALESCE(s.cost, 0.0) as cost,
			COALESCE(p.name, '') as project_name,
			(SELECT COUNT(*) FROM %s WHERE session_id = s.id) as message_count
		FROM %s s
		LEFT JOIN project p ON s.project_id = p.id
		WHERE s.parent_id IS NULL
		ORDER BY s.time_updated DESC
		LIMIT ?
	`, c.messagesTable, c.sessionsTable)

	rows, err := c.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var updatedAt, createdAt int64

		err := rows.Scan(
			&s.ID,
			&s.Title,
			&s.Directory,
			&s.ProjectID,
			&s.Slug,
			&updatedAt,
			&createdAt,
			&s.Agent,
			&s.Model,
			&s.Cost,
			&s.ProjectName,
			&s.MessageCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		s.UpdatedAt = time.Unix(updatedAt/1000, 0)
		s.CreatedAt = time.Unix(createdAt/1000, 0)
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (c *Client) GetSessionByID(id string) (*models.Session, error) {
	query := fmt.Sprintf(`
		SELECT 
			id,
			COALESCE(title, 'Untitled Session') as title,
			directory,
			project_id,
			slug,
			time_updated,
			time_created
		FROM %s
		WHERE id = ? OR id LIKE ? OR SUBSTR(id, 1, ?) = ?
		ORDER BY time_updated DESC
		LIMIT 1
	`, c.sessionsTable)

	var s models.Session
	var updatedAt, createdAt int64

	// Try exact match, then prefix match, then substring match for short ID
	err := c.db.QueryRow(query, id, id+"%", len(id), id).Scan(
		&s.ID,
		&s.Title,
		&s.Directory,
		&s.ProjectID,
		&s.Slug,
		&updatedAt,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	s.UpdatedAt = time.Unix(updatedAt/1000, 0)
	s.CreatedAt = time.Unix(createdAt/1000, 0)

	return &s, nil
}

func (c *Client) SearchSessions(query string, limit int) ([]models.Session, error) {
	searchQuery := fmt.Sprintf(`
		SELECT 
			id,
			COALESCE(title, 'Untitled Session') as title,
			directory,
			project_id,
			slug,
			time_updated,
			time_created
		FROM %s
		WHERE parent_id IS NULL AND (title LIKE ? OR id LIKE ?)
		ORDER BY time_updated DESC
		LIMIT ?
	`, c.sessionsTable)

	searchPattern := "%" + query + "%"
	rows, err := c.db.Query(searchQuery, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var updatedAt, createdAt int64

		err := rows.Scan(
			&s.ID,
			&s.Title,
			&s.Directory,
			&s.ProjectID,
			&s.Slug,
			&updatedAt,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		s.UpdatedAt = time.Unix(updatedAt/1000, 0)
		s.CreatedAt = time.Unix(createdAt/1000, 0)
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (c *Client) GetMessageCount(sessionID string) (int, error) {
	var count int
	err := c.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE session_id = ?", c.messagesTable), sessionID).Scan(&count)
	return count, err
}

func (c *Client) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalSessions int
	err := c.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE parent_id IS NULL", c.sessionsTable)).Scan(&totalSessions)
	if err != nil {
		return nil, err
	}
	stats["total_sessions"] = totalSessions

	var totalMessages int
	err = c.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", c.messagesTable)).Scan(&totalMessages)
	if err != nil {
		return nil, err
	}
	stats["total_messages"] = totalMessages

	return stats, nil
}

func (c *Client) DeleteSession(sessionID string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete message children explicitly: SQLite foreign keys are not
	// guaranteed to be enabled on this connection, and cascades would leave
	// orphaned rows behind. In v2 storage, session_part references
	// session_message, so parts go first.
	if c.sessionsTable == "session_v2" {
		if _, err := tx.Exec("DELETE FROM session_part WHERE session_id = ?", sessionID); err != nil {
			return fmt.Errorf("failed to delete session parts: %w", err)
		}
		if _, err := tx.Exec("DELETE FROM session_message WHERE session_id = ?", sessionID); err != nil {
			return fmt.Errorf("failed to delete session messages: %w", err)
		}
	} else {
		if _, err := tx.Exec("DELETE FROM message WHERE session_id = ?", sessionID); err != nil {
			return fmt.Errorf("failed to delete messages: %w", err)
		}
	}

	res, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", c.sessionsTable), sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return tx.Commit()
}

func GetDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	defaultPath := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}

	return "", fmt.Errorf("opencode database not found at %s", defaultPath)
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
