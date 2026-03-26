package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Theme string `json:"theme"`

	LastGroup int `json:"last_group"`

	ShowPreview bool `json:"show_preview"`

	PreviewWidth int `json:"preview_width"`

	SortBy string `json:"sort_by"`
}

var DefaultConfig = Config{
	Theme:        "default",
	LastGroup:    0,
	ShowPreview:  true,
	PreviewWidth: 40,
	SortBy:       "date",
}

func Load() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig
			if err := Save(&cfg); err != nil {
				return nil, err
			}
			return &cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Theme == "" {
		cfg.Theme = DefaultConfig.Theme
	}
	if cfg.PreviewWidth == 0 {
		cfg.PreviewWidth = DefaultConfig.PreviewWidth
	}
	if cfg.SortBy == "" {
		cfg.SortBy = DefaultConfig.SortBy
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return nil
	}

	return os.WriteFile(path, data, 0644)
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "osem", "config.json"), nil
}
