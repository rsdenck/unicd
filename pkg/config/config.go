package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Context struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Org        string `json:"org"`
	VDC        string `json:"vdc,omitempty"`
	Username   string `json:"username"`
	Token      string `json:"token,omitempty"`
	APIVersion string `json:"apiVersion"`
}

type Config struct {
	CurrentContext string    `json:"currentContext"`
	Contexts       []Context `json:"contexts"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".unicd")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func sessionPath() string {
	return filepath.Join(configDir(), "session.json")
}

func Load() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Contexts: []Context{}}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func GetCurrentContext() (*Context, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	for _, ctx := range cfg.Contexts {
		if ctx.Name == cfg.CurrentContext {
			return &ctx, nil
		}
	}
	return nil, fmt.Errorf("no context set")
}

func SaveSession(session map[string]string) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, _ := json.Marshal(session)
	return os.WriteFile(sessionPath(), data, 0600)
}

func LoadSession() (map[string]string, error) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return nil, err
	}
	var session map[string]string
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return session, nil
}

func ClearSession() error {
	return os.Remove(sessionPath())
}
