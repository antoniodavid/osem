package models

import "os"

type Config struct {
	DBPath           string `json:"db_path"`
	TmuxPrefix       string `json:"tmux_prefix"`
	DefaultDirectory string `json:"default_directory"`
	PreviewLines     int    `json:"preview_lines"`
	Theme            string `json:"theme"`
}

func DefaultConfig() *Config {
	return &Config{
		DBPath:           "~/.local/share/opencode/opencode.db",
		TmuxPrefix:       "opencode-",
		DefaultDirectory: "~",
		PreviewLines:     10,
		Theme:            "default",
	}
}

func (c *Config) GetDBPath() string {
	return expandHome(c.DBPath)
}

func (c *Config) GetDefaultDirectory() string {
	return expandHome(c.DefaultDirectory)
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := getHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}

func getHomeDir() (string, error) {
	return os.UserHomeDir()
}
