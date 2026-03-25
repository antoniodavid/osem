package models

import "time"

type Session struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Directory    string    `json:"directory"`
	ProjectID    string    `json:"project_id"`
	Slug         string    `json:"slug"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedAt    time.Time `json:"created_at"`
	MessageCount int       `json:"message_count"`
	IsFavorite   bool      `json:"is_favorite"`
	TmuxSession  *string   `json:"tmux_session,omitempty"`
}

type SessionWithMetadata struct {
	Session
	IsFavorite bool   `json:"is_favorite"`
	Alias      string `json:"alias,omitempty"`
}

func (s *Session) ShortID() string {
	if len(s.ID) > 16 {
		return s.ID[:12]
	}
	return s.ID
}

func (s *Session) DisplayName() string {
	if s.Title != "" {
		return s.Title
	}
	return "Untitled Session"
}

func (s *Session) TmuxName(prefix string) string {
	return prefix + s.ShortID()
}
