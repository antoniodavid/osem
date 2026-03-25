package bookmarks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/adruban/osem/internal/models"
)

type Manager struct {
	configPath string
	config     models.BookmarksConfig
	mu         sync.RWMutex
}

func NewManager() (*Manager, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	m := &Manager{
		configPath: configPath,
	}

	if err := m.load(); err != nil {
		// Create new config if doesn't exist
		m.config = models.BookmarksConfig{Bookmarks: []models.Bookmark{}}
	}

	return m, nil
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", "osem")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "bookmarks.json"), nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.config)
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) Add(sessionID, alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Add(sessionID, alias)
	return m.save()
}

func (m *Manager) Remove(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Remove(sessionID)
	return m.save()
}

func (m *Manager) IsBookmarked(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.Contains(sessionID)
}

func (m *Manager) GetAlias(sessionID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.GetAlias(sessionID)
}

func (m *Manager) GetAll() []models.Bookmark {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.Bookmarks
}

func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, len(m.config.Bookmarks))
	for i, bm := range m.config.Bookmarks {
		ids[i] = bm.SessionID
	}
	return ids
}
