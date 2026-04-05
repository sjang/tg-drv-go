package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	TelegramAPIID   int    `json:"telegram_api_id"`
	TelegramAPIHash string `json:"telegram_api_hash"`
	DBPath          string `json:"db_path"`
	HTTPPort        int    `json:"http_port"`
	DataDir         string `json:"data_dir"`
}

func DefaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tg-drv")
}

func DefaultConfig() *Config {
	dataDir := DefaultDataDir()
	return &Config{
		DBPath:  filepath.Join(dataDir, "tgdrv.db"),
		HTTPPort: 9876,
		DataDir: dataDir,
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) EnsureDataDir() error {
	return os.MkdirAll(c.DataDir, 0755)
}
