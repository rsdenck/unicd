package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type TomlConfig struct {
	Org  string `toml:"org"`
	VDC  string `toml:"vdc"`
	User string `toml:"user"`
	Pass string `toml:"pass"`
	Host string `toml:"host"`
}

func tomlPath() string {
	return filepath.Join(configDir(), "config.toml")
}

func LoadToml() (*TomlConfig, error) {
	path := tomlPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg TomlConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config.toml: %w", err)
	}
	return &cfg, nil
}

func SaveToml(cfg *TomlConfig) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	f, err := os.Create(tomlPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
