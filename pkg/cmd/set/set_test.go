package set

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestSetCommand_Todo(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "* DONE Task") {
		t.Errorf("TODO should be changed to DONE, got: %s", string(updated))
	}
}

func TestSetCommand_Tags(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading :old:
:PROPERTIES:
:ID: test-123
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--tags", "+new,-old", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if strings.Contains(string(updated), ":old:") {
		t.Errorf("old tag should be removed, got: %s", string(updated))
	}
	if !strings.Contains(string(updated), ":new:") {
		t.Errorf("new tag should be added, got: %s", string(updated))
	}
}

func TestSetCommand_Title(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Old Title
:PROPERTIES:
:ID: test-123
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--title", "New Title", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "New Title") {
		t.Errorf("title should be changed, got: %s", string(updated))
	}
}

func TestSetCommand_MarkdownTitle_WithID(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	content := `# Old Title
<!-- orgx-id: test-id-123 -->

Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-id-123", "--title", "New Title", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "# New Title") {
		t.Errorf("markdown title should be changed, got: %s", string(updated))
	}
	if !strings.Contains(string(updated), "<!-- orgx-id: test-id-123 -->") {
		t.Errorf("orgx-id marker should be preserved, got: %s", string(updated))
	}
}

func TestSetCommand_MarkdownTitle_RequiresStableRef(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	content := `# Heading
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::H:1234", "--title", "New Title", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for unstable markdown ref")
	}
}

func TestSetCommand_DryRun(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "DONE", "--dry-run"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Changes to apply") {
		t.Errorf("dry-run should show changes, got: %s", output)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "* TODO Task") {
		t.Errorf("file should not be modified in dry-run, got: %s", string(updated))
	}
}

func TestSetCommand_NoChanges(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* DONE Task
:PROPERTIES:
:ID: test-123
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No changes") {
		t.Errorf("should report no changes, got: %s", output)
	}
}

func TestSetCommand_Backup(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)
	cmd.Execute()

	files, _ := filepath.Glob(filepath.Join(tmpDir, "test.org~*"))
	if len(files) != 1 {
		t.Errorf("expected 1 backup file, got %d", len(files))
	}
}

func TestSetCommand_NotFound(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:nonexistent", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent ref")
	}
}

func TestSetCommand_FileNotFound(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{"/nonexistent/file.org::ID:test", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSetCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	called := false
	cmd := NewCmdSet(f, func(opts *SetOptions) error {
		called = true
		if opts.Todo != "DONE" {
			t.Errorf("todo = %q, want DONE", opts.Todo)
		}
		return nil
	})
	cmd.SetArgs([]string{"path.org::ID:test", "--todo", "DONE"})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}

func TestComputeNewTags(t *testing.T) {
	tests := []struct {
		name    string
		old     []string
		ops     []string
		want    []string
		wantLen int
	}{
		{
			name:    "add tag",
			old:     []string{"a"},
			ops:     []string{"+b"},
			wantLen: 2,
		},
		{
			name:    "remove tag",
			old:     []string{"a", "b"},
			ops:     []string{"-a"},
			wantLen: 1,
		},
		{
			name:    "replace all",
			old:     []string{"a", "b"},
			ops:     []string{"c"},
			wantLen: 1,
		},
		{
			name:    "add and remove",
			old:     []string{"a", "b"},
			ops:     []string{"+c", "-a"},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeNewTags(tt.old, tt.ops)
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestEqualTags(t *testing.T) {
	tests := []struct {
		a, b []string
		want bool
	}{
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"b", "a"}, true},
		{[]string{"a"}, []string{"a", "b"}, false},
		{[]string{}, []string{}, true},
		{nil, nil, true},
	}

	for _, tt := range tests {
		got := equalTags(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("equalTags(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestReplaceTodoInLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		oldTodo string
		newTodo string
		want    string
	}{
		{
			name:    "replace existing",
			line:    "* TODO Task",
			oldTodo: "TODO",
			newTodo: "DONE",
			want:    "* DONE Task",
		},
		{
			name:    "add todo",
			line:    "* Task",
			oldTodo: "",
			newTodo: "TODO",
			want:    "* TODO Task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceTodoInLine(tt.line, tt.oldTodo, tt.newTodo)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetCommand_StateLogging(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "STRT", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), ":LOGBOOK:") {
		t.Error("LOGBOOK drawer should be created")
	}
	if !strings.Contains(string(updated), "State \"STRT\"") {
		t.Error("state change should be logged")
	}
	if !strings.Contains(string(updated), "from \"TODO\"") {
		t.Error("old state should be in log")
	}
}

func TestSetCommand_StateLogging_NoLog(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "DONE", "--no-log", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if strings.Contains(string(updated), ":LOGBOOK:") {
		t.Error("LOGBOOK should not be created with --no-log")
	}
}

func TestSetCommand_Scheduled(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--scheduled", "2026-01-15", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "SCHEDULED:") {
		t.Error("SCHEDULED should be added")
	}
	if !strings.Contains(string(updated), "2026-01-15") {
		t.Error("date should be in SCHEDULED")
	}
}

func TestSetCommand_Deadline(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--deadline", "+3d", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "DEADLINE:") {
		t.Error("DEADLINE should be added")
	}
}

func TestSetCommand_AutoClose(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-123", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "CLOSED:") {
		t.Error("CLOSED should be auto-added for DONE state")
	}
}

func TestSetCommand_Created(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::/Heading", "--created", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), ":CREATED:") {
		t.Error("CREATED property should be added")
	}
	if !strings.Contains(string(updated), ":PROPERTIES:") {
		t.Error("PROPERTIES drawer should be created")
	}
}

func TestSetCommand_AutoClose_PreservesIDRef(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task
:PROPERTIES:
:ID: preserve-me-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:preserve-me-123", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("first set failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "CLOSED:") {
		t.Fatal("CLOSED should be added")
	}

	// Key test: After adding CLOSED, can we still resolve the ID ref?
	ios2, _, stdout2, _ := iostreams.Test()
	f2 := &cmdutil.Factory{IOStreams: ios2}

	cmd2 := NewCmdSet(f2, nil)
	cmd2.SetArgs([]string{path + "::ID:preserve-me-123", "--title", "Modified Task", "--yes"})
	cmd2.SetOut(stdout2)

	err = cmd2.Execute()
	if err != nil {
		t.Fatalf("second set by ID failed after CLOSED was added: %v", err)
	}

	final, _ := os.ReadFile(path)
	if !strings.Contains(string(final), "Modified Task") {
		t.Errorf("title should be changed, got: %s", string(final))
	}
}

func TestSetCommand_TodoDonePreservesDeadline(t *testing.T) {
	// When changing state to DONE, existing DEADLINE should be preserved
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task with deadline
SCHEDULED: <2026-01-15 Thu> DEADLINE: <2026-01-20 Tue>
:PROPERTIES:
:ID: deadline-test-123
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdSet(f, nil)
	cmd.SetArgs([]string{path + "::ID:deadline-test-123", "--todo", "DONE", "--yes"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	updated, _ := os.ReadFile(path)
	updatedStr := string(updated)

	// DONE state should be set
	if !strings.Contains(updatedStr, "* DONE") {
		t.Error("TODO should be changed to DONE")
	}

	// CLOSED should be added
	if !strings.Contains(updatedStr, "CLOSED:") {
		t.Error("CLOSED should be added for DONE state")
	}

	// DEADLINE must be preserved (this is the key test)
	if !strings.Contains(updatedStr, "DEADLINE:") {
		t.Errorf("DEADLINE should be preserved, got: %s", updatedStr)
	}

	// SCHEDULED should also be preserved
	if !strings.Contains(updatedStr, "SCHEDULED:") {
		t.Errorf("SCHEDULED should be preserved, got: %s", updatedStr)
	}
}
