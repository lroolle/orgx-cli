package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version          int                  `yaml:"version"`
	DefaultWorkspace string               `yaml:"default_workspace"`
	Workspaces       map[string]Workspace `yaml:"workspaces"`
	Timestamps       TimestampConfig      `yaml:"timestamps,omitempty"`
	States           StatesConfig         `yaml:"states,omitempty"`
	Capture          CaptureConfig        `yaml:"capture,omitempty"`
}

type TimestampConfig struct {
	Timezone string `yaml:"timezone,omitempty"`
}

type StatesConfig struct {
	Keywords   []string `yaml:"keywords,omitempty"`
	DoneStates []string `yaml:"done_states,omitempty"`
	AutoClose  bool     `yaml:"auto_close,omitempty"`
}

type CaptureConfig struct {
	DefaultFile  string `yaml:"default_file,omitempty"`
	DefaultState string `yaml:"default_state,omitempty"`
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

var configWarnOnce sync.Once

func LoadOrDefault() *Config {
	cfg, err := Load()
	if err != nil {
		if !os.IsNotExist(err) {
			configWarnOnce.Do(func() {
				fmt.Fprintf(os.Stderr, "warning: config load error, using defaults: %v\n", err)
			})
		}
		return &Config{
			Version:    1,
			Workspaces: make(map[string]Workspace),
		}
	}
	if cfg == nil {
		return &Config{
			Version:    1,
			Workspaces: make(map[string]Workspace),
		}
	}
	return cfg
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

var defaultDoneStates = []string{"DONE", "KILL", "CANCELLED"}

func (c *Config) IsDoneState(state string) bool {
	states := c.States.DoneStates
	if len(states) == 0 {
		states = defaultDoneStates
	}
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func (c *Config) ShouldAutoClose() bool {
	if len(c.States.DoneStates) == 0 && !c.States.AutoClose {
		return true
	}
	return c.States.AutoClose
}

func (c *Config) GetCaptureFile() string {
	if c.Capture.DefaultFile != "" {
		return c.Capture.DefaultFile
	}
	return "INBOX.org"
}

func (c *Config) GetCaptureState() string {
	if c.Capture.DefaultState != "" {
		return c.Capture.DefaultState
	}
	return "IDEA"
}

func (c *Config) GetTimezone() *time.Location {
	if c.Timestamps.Timezone != "" {
		loc, err := time.LoadLocation(c.Timestamps.Timezone)
		if err == nil {
			return loc
		}
	}
	return time.Local
}

var defaultStateKeywords = []string{
	"IDEA", "TODO", "PROJ", "LOOP", "STRT", "WAIT", "HOLD", "DONE", "KILL", "CANCELLED", "NEXT",
}

func (c *Config) GetStateKeywords() []string {
	if len(c.States.Keywords) > 0 {
		return c.States.Keywords
	}
	return defaultStateKeywords
}
