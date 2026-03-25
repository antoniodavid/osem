package models

import "time"

type Bookmark struct {
	SessionID string    `json:"session_id"`
	Alias     string    `json:"alias"`
	AddedAt   time.Time `json:"added_at"`
}

type BookmarksConfig struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

func (b *BookmarksConfig) Contains(sessionID string) bool {
	for _, bm := range b.Bookmarks {
		if bm.SessionID == sessionID {
			return true
		}
	}
	return false
}

func (b *BookmarksConfig) Add(sessionID, alias string) {
	b.Bookmarks = append(b.Bookmarks, Bookmark{
		SessionID: sessionID,
		Alias:     alias,
		AddedAt:   time.Now(),
	})
}

func (b *BookmarksConfig) Remove(sessionID string) {
	for i, bm := range b.Bookmarks {
		if bm.SessionID == sessionID {
			b.Bookmarks = append(b.Bookmarks[:i], b.Bookmarks[i+1:]...)
			return
		}
	}
}

func (b *BookmarksConfig) GetAlias(sessionID string) string {
	for _, bm := range b.Bookmarks {
		if bm.SessionID == sessionID {
			return bm.Alias
		}
	}
	return ""
}
