package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version          int                  `yaml:"version"`
	DefaultWorkspace string               `yaml:"default_workspace"`
	Workspaces       map[string]Workspace `yaml:"workspaces"`
}

type Workspace struct {
	Root   string `yaml:"root"`
	RoamDB string `yaml:"roam_db,omitempty"`
	Inbox  string `yaml:"inbox,omitempty"`
	Format string `yaml:"format,omitempty"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "orgx", "config.yaml")
}

func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				Version:    1,
				Workspaces: make(map[string]Workspace),
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Workspaces == nil {
		cfg.Workspaces = make(map[string]Workspace)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	return c.SaveTo(DefaultConfigPath())
}

func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) GetWorkspace(name string) (*Workspace, error) {
	if name == "" {
		name = c.DefaultWorkspace
	}
	if name == "" {
		return nil, fmt.Errorf("no workspace specified and no default set")
	}

	ws, ok := c.Workspaces[name]
	if !ok {
		return nil, fmt.Errorf("workspace not found: %s", name)
	}
	return &ws, nil
}

func (c *Config) AddWorkspace(name string, ws Workspace) error {
	if _, exists := c.Workspaces[name]; exists {
		return fmt.Errorf("workspace already exists: %s", name)
	}
	c.Workspaces[name] = ws
	return nil
}

func (c *Config) SetDefault(name string) error {
	if _, exists := c.Workspaces[name]; !exists {
		return fmt.Errorf("workspace not found: %s", name)
	}
	c.DefaultWorkspace = name
	return nil
}
