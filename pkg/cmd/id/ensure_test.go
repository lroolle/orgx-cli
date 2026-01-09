package id

import (
	"os"
	"path/filepath"
	"regexp"
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

func TestEnsure_AddsIDToMarkdownHeading(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	content := `# Heading Without ID

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
	lines := strings.Split(string(updated), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	re := regexp.MustCompile(`^<!--\s*orgx-id:\s*[a-f0-9-]{36}\s*-->$`)
	if !re.MatchString(strings.TrimSpace(lines[1])) {
		t.Fatalf("expected orgx-id marker on line 2, got %q", lines[1])
	}
}

func TestEnsure_PreservesExistingMarkdownID(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	content := `# Heading With ID
<!-- orgx-id: existing-id-123 -->

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
	if strings.Count(string(updated), "orgx-id:") != 1 {
		t.Error("should not add duplicate orgx-id marker")
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

func TestEnsure_SetextHeading(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	// Setext H1 (=====) and H2 (-----)
	content := `Setext H1 Title
===============

Body under H1.

Setext H2 Title
---------------

Body under H2.
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
	lines := strings.Split(string(updated), "\n")

	// Find the setext underlines and verify marker is AFTER them, not before
	re := regexp.MustCompile(`^<!--\s*orgx-id:\s*[a-f0-9-]{36}\s*-->$`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check H1 underline (=====)
		if len(trimmed) > 0 && trimmed == strings.Repeat("=", len(trimmed)) {
			// Line before should be title, not marker
			if i > 0 && re.MatchString(strings.TrimSpace(lines[i-1])) {
				t.Errorf("orgx-id marker inserted BEFORE setext underline (corrupted): line %d", i)
			}
			// Line after should be marker (or blank then marker)
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if nextLine != "" && !re.MatchString(nextLine) {
					// Could be blank line, check next
					if i+2 < len(lines) && !re.MatchString(strings.TrimSpace(lines[i+2])) {
						t.Errorf("orgx-id marker not found after H1 setext underline")
					}
				}
			}
		}

		// Check H2 underline (-----)
		if len(trimmed) > 0 && trimmed == strings.Repeat("-", len(trimmed)) && len(trimmed) >= 3 {
			// Line before should be title, not marker
			if i > 0 && re.MatchString(strings.TrimSpace(lines[i-1])) {
				t.Errorf("orgx-id marker inserted BEFORE setext underline (corrupted): line %d", i)
			}
		}
	}

	// Verify both headings got IDs
	markerCount := 0
	for _, line := range lines {
		if re.MatchString(strings.TrimSpace(line)) {
			markerCount++
		}
	}
	if markerCount != 2 {
		t.Errorf("expected 2 orgx-id markers for 2 setext headings, got %d", markerCount)
	}

	// Run ensure again - should be idempotent (no new changes)
	cmd2 := NewCmdEnsure(f, nil)
	cmd2.SetArgs([]string{path, "--yes"})
	cmd2.SetOut(stdout)

	err = cmd2.Execute()
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	updated2, _ := os.ReadFile(path)
	markerCount2 := 0
	for _, line := range strings.Split(string(updated2), "\n") {
		if re.MatchString(strings.TrimSpace(line)) {
			markerCount2++
		}
	}
	if markerCount2 != 2 {
		t.Errorf("idempotency failed: expected 2 markers after second run, got %d", markerCount2)
	}
}
