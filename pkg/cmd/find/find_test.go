package find

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestFindCommand_ByQuery(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Project Alpha
* Project Beta
* Something Else
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"Project", "--in", tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Project Alpha") {
		t.Errorf("output should contain 'Project Alpha', got: %s", output)
	}
	if !strings.Contains(output, "Project Beta") {
		t.Errorf("output should contain 'Project Beta', got: %s", output)
	}
	if strings.Contains(output, "Something Else") {
		t.Errorf("output should not contain 'Something Else', got: %s", output)
	}
}

func TestFindCommand_ByTodo(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task 1
* DONE Task 2
* TODO Task 3
* Just a heading
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"", "--todo", "TODO", "--in", tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 TODO matches, got %d: %s", len(lines), output)
	}
}

func TestFindCommand_ByTag(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Work task :work:
* Personal task :personal:
* Work and personal :work:personal:
* No tags
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"", "--tag", "work", "--in", tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 work tag matches, got %d: %s", len(lines), output)
	}
}

func TestFindCommand_JSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Task :tag1:
:PROPERTIES:
:ID: test-123
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"Task", "--in", tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var results []FindResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Task" {
		t.Errorf("title = %q, want Task", r.Title)
	}
	if r.Todo != "TODO" {
		t.Errorf("todo = %q, want TODO", r.Todo)
	}
	if len(r.Tags) != 1 || r.Tags[0] != "tag1" {
		t.Errorf("tags = %v, want [tag1]", r.Tags)
	}
}

func TestFindCommand_Limit(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading 1
* Heading 2
* Heading 3
* Heading 4
* Heading 5
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"Heading", "--in", tmpDir, "--limit", "3", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var results []FindResult
	json.Unmarshal(stdout.Bytes(), &results)

	if len(results) != 3 {
		t.Errorf("expected 3 results with limit, got %d", len(results))
	}
}

func TestFindCommand_MultipleFiles(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.org"), []byte("* Match 1\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.org"), []byte("* Match 2\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.org"), []byte("* Other\n"), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"Match", "--in", tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var results []FindResult
	json.Unmarshal(stdout.Bytes(), &results)

	if len(results) != 2 {
		t.Errorf("expected 2 results from 2 files, got %d", len(results))
	}
}

func TestFindCommand_NoMatches(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"nonexistent", "--in", tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No matches") {
		t.Errorf("expected 'No matches' message, got: %s", output)
	}
}

func TestFindCommand_GlobPattern(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "notes.org"), []byte("* Note\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "tasks.org"), []byte("* Task\n"), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"", "--in", filepath.Join(tmpDir, "notes.org"), "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var results []FindResult
	json.Unmarshal(stdout.Bytes(), &results)

	if len(results) != 1 {
		t.Errorf("expected 1 result from specific file, got %d", len(results))
	}
}

func TestFindCommand_CombinedFilters(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	content := `* TODO Work task :work:
* TODO Personal task :personal:
* DONE Work done :work:
* Work heading :work:
`
	os.WriteFile(filepath.Join(tmpDir, "test.org"), []byte(content), 0644)

	cmd := NewCmdFind(f, nil)
	cmd.SetArgs([]string{"Work", "--todo", "TODO", "--tag", "work", "--in", tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var results []FindResult
	json.Unmarshal(stdout.Bytes(), &results)

	if len(results) != 1 {
		t.Errorf("expected 1 result with all filters, got %d", len(results))
	}
}

func TestFindCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	called := false
	cmd := NewCmdFind(f, func(opts *FindOptions) error {
		called = true
		if opts.Query != "test" {
			t.Errorf("query = %q, want test", opts.Query)
		}
		return nil
	})
	cmd.SetArgs([]string{"test", "--in", "."})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}

func TestFindFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.org"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.org"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.txt"), []byte(""), 0644)

	files, err := findFiles(tmpDir)
	if err != nil {
		t.Fatalf("findFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 org files, got %d", len(files))
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"path.org::ID:abc", "path.org"},
		{"path.org::/Outline", "path.org"},
		{"/full/path.org::H:hash", "/full/path.org"},
		{"path.org", "path.org"},
	}

	for _, tt := range tests {
		got := extractPath(tt.ref)
		if got != tt.want {
			t.Errorf("extractPath(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}
