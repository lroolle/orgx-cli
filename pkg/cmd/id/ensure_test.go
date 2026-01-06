package id

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestEnsure_AddsIDToHeading(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading Without ID
Body text.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path, "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), ":ID:") {
		t.Error("ID should be added")
	}
	if !strings.Contains(string(updated), ":PROPERTIES:") {
		t.Error("PROPERTIES drawer should be added")
	}
	if !strings.Contains(string(updated), ":END:") {
		t.Error("END should be added")
	}
}

func TestEnsure_PreservesExistingID(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading With ID
:PROPERTIES:
:ID: existing-id-123
:END:
Body text.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path, "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "existing-id-123") {
		t.Error("existing ID should be preserved")
	}
	if strings.Count(string(updated), ":ID:") != 1 {
		t.Error("should not add duplicate ID")
	}
}

func TestEnsure_DryRun(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading Without ID
Body text.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path, "--dry-run"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if strings.Contains(string(updated), ":ID:") {
		t.Error("dry-run should not modify file")
	}
}

func TestEnsure_RequiresConfirmation(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	ios.SetStdinTTY(false)
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading Without ID
Body text.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("should require --yes in non-interactive mode")
	}
}

func TestEnsure_FilterByTodo(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
Body.
* Regular Heading
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path, "--todo", "TODO", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	lines := strings.Split(string(updated), "\n")

	todoHasID := false
	regularHasID := false
	for i, line := range lines {
		if strings.Contains(line, "TODO Task") {
			if i+1 < len(lines) && strings.Contains(lines[i+1], ":PROPERTIES:") {
				todoHasID = true
			}
		}
		if strings.Contains(line, "Regular Heading") {
			if i+1 < len(lines) && strings.Contains(lines[i+1], ":PROPERTIES:") {
				regularHasID = true
			}
		}
	}

	if !todoHasID {
		t.Error("TODO heading should have ID")
	}
	if regularHasID {
		t.Error("Regular heading should not have ID (filtered out)")
	}
}

func TestEnsure_AddsToExistingDrawer(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
:PROPERTIES:
:CUSTOM_ID: custom-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path, "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), ":ID:") {
		t.Error("ID should be added")
	}
	if !strings.Contains(string(updated), ":CUSTOM_ID: custom-123") {
		t.Error("existing property should be preserved")
	}
	if strings.Count(string(updated), ":PROPERTIES:") != 1 {
		t.Error("should not create duplicate PROPERTIES drawer")
	}
}

func TestEnsure_MultipleHeadings(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* First Heading
Body.
* Second Heading
Body.
* Third Heading
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdEnsure(f, nil)
	cmd.SetArgs([]string{path, "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	idCount := strings.Count(string(updated), ":ID:")
	if idCount != 3 {
		t.Errorf("expected 3 IDs, got %d", idCount)
	}
}
