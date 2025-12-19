package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Version:          1,
		DefaultWorkspace: "work",
		Workspaces: map[string]Workspace{
			"work": {
				Root:   "/home/user/org",
				RoamDB: "/home/user/.emacs.d/org-roam.db",
				Inbox:  "/home/user/org/inbox.org",
			},
			"notes": {
				Root:   "/home/user/notes",
				Format: "md",
			},
		},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if loaded.Version != cfg.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, cfg.Version)
	}

	if loaded.DefaultWorkspace != cfg.DefaultWorkspace {
		t.Errorf("DefaultWorkspace = %q, want %q", loaded.DefaultWorkspace, cfg.DefaultWorkspace)
	}

	if len(loaded.Workspaces) != 2 {
		t.Errorf("len(Workspaces) = %d, want 2", len(loaded.Workspaces))
	}

	ws, err := loaded.GetWorkspace("work")
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}

	if ws.Root != "/home/user/org" {
		t.Errorf("Root = %q, want /home/user/org", ws.Root)
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.yaml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}

	if len(cfg.Workspaces) != 0 {
		t.Errorf("len(Workspaces) = %d, want 0", len(cfg.Workspaces))
	}
}

func TestAddWorkspace(t *testing.T) {
	cfg := &Config{
		Version:    1,
		Workspaces: make(map[string]Workspace),
	}

	ws := Workspace{Root: "/tmp/test"}
	if err := cfg.AddWorkspace("test", ws); err != nil {
		t.Fatalf("AddWorkspace() error = %v", err)
	}

	if len(cfg.Workspaces) != 1 {
		t.Errorf("len(Workspaces) = %d, want 1", len(cfg.Workspaces))
	}

	err := cfg.AddWorkspace("test", ws)
	if err == nil {
		t.Error("expected error for duplicate workspace")
	}
}

func TestSetDefault(t *testing.T) {
	cfg := &Config{
		Version:    1,
		Workspaces: map[string]Workspace{
			"work": {Root: "/work"},
		},
	}

	if err := cfg.SetDefault("work"); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}

	if cfg.DefaultWorkspace != "work" {
		t.Errorf("DefaultWorkspace = %q, want work", cfg.DefaultWorkspace)
	}

	err := cfg.SetDefault("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestGetWorkspace(t *testing.T) {
	cfg := &Config{
		Version:          1,
		DefaultWorkspace: "work",
		Workspaces: map[string]Workspace{
			"work": {Root: "/work"},
		},
	}

	ws, err := cfg.GetWorkspace("")
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}

	if ws.Root != "/work" {
		t.Errorf("Root = %q, want /work", ws.Root)
	}

	_, err = cfg.GetWorkspace("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace")
	}

	cfg.DefaultWorkspace = ""
	_, err = cfg.GetWorkspace("")
	if err == nil {
		t.Error("expected error when no default set")
	}
}

func TestConfigYAMLRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Version:          1,
		DefaultWorkspace: "test",
		Workspaces: map[string]Workspace{
			"test": {
				Root:   "/test",
				RoamDB: "/test/roam.db",
			},
		},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := `version: 1
default_workspace: test
workspaces:
    test:
        root: /test
        roam_db: /test/roam.db
`
	if string(data) != expected {
		t.Errorf("YAML output:\n%s\nwant:\n%s", string(data), expected)
	}
}
