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
}

func NewClient(dbPath string) (*Client, error) {
	expandedPath := expandHome(dbPath)
	db, err := sql.Open("sqlite", expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &Client{db: db}, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) ListSessions(limit int) ([]models.Session, error) {
	query := `
		SELECT 
			id,
			COALESCE(title, 'Untitled Session') as title,
			directory,
			project_id,
			slug,
			time_updated,
			time_created
		FROM session
		WHERE parent_id IS NULL
		ORDER BY time_updated DESC
		LIMIT ?
	`

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
	query := `
		SELECT 
			id,
			COALESCE(title, 'Untitled Session') as title,
			directory,
			project_id,
			slug,
			time_updated,
			time_created
		FROM session
		WHERE id = ? OR id LIKE ? OR SUBSTR(id, 1, ?) = ?
		ORDER BY time_updated DESC
		LIMIT 1
	`

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
	searchQuery := `
		SELECT 
			id,
			COALESCE(title, 'Untitled Session') as title,
			directory,
			project_id,
			slug,
			time_updated,
			time_created
		FROM session
		WHERE parent_id IS NULL AND (title LIKE ? OR id LIKE ?)
		ORDER BY time_updated DESC
		LIMIT ?
	`

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
	err := c.db.QueryRow("SELECT COUNT(*) FROM message WHERE session_id = ?", sessionID).Scan(&count)
	return count, err
}

func (c *Client) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalSessions int
	err := c.db.QueryRow("SELECT COUNT(*) FROM session WHERE parent_id IS NULL").Scan(&totalSessions)
	if err != nil {
		return nil, err
	}
	stats["total_sessions"] = totalSessions

	var totalMessages int
	err = c.db.QueryRow("SELECT COUNT(*) FROM message").Scan(&totalMessages)
	if err != nil {
		return nil, err
	}
	stats["total_messages"] = totalMessages

	return stats, nil
}

func (c *Client) DeleteSession(sessionID string) error {
	_, err := c.db.Exec("DELETE FROM session WHERE id = ?", sessionID)
	return err
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
