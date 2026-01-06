package get

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
)

func TestGetCommand_ByID(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading 1
:PROPERTIES:
:ID: test-id-123
:END:
Body content here.

** Subheading
Sub content.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::ID:test-id-123"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Heading 1") {
		t.Errorf("output should contain heading title, got: %s", output)
	}
}

func TestGetCommand_ByOutline(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* My Heading
Body content.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::/My Heading"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "My Heading") {
		t.Errorf("output should contain heading, got: %s", output)
	}
}

func TestGetCommand_JSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Heading :tag1:
:PROPERTIES:
:ID: json-test
:END:
Body.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::ID:json-test", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var h ir.Heading
	if err := json.Unmarshal(stdout.Bytes(), &h); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if h.Title != "Heading" {
		t.Errorf("title = %q, want %q", h.Title, "Heading")
	}
	if h.Todo != "TODO" {
		t.Errorf("todo = %q, want TODO", h.Todo)
	}
	if len(h.Tags) != 1 || h.Tags[0] != "tag1" {
		t.Errorf("tags = %v, want [tag1]", h.Tags)
	}
}

func TestGetCommand_NoChildren(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Parent
:PROPERTIES:
:ID: parent-id
:END:
** Child
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::ID:parent-id", "--no-children", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var h ir.Heading
	json.Unmarshal(stdout.Bytes(), &h)

	if len(h.Children) != 0 {
		t.Errorf("expected no children with --no-children, got %d", len(h.Children))
	}
}

func TestGetCommand_FormatMd(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* TODO Heading :tag:
:PROPERTIES:
:ID: format-test
:END:
Body content.
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::ID:format-test", "--format", "md"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.HasPrefix(output, "#") {
		t.Errorf("markdown format should start with #, got: %s", output)
	}
}

func TestGetCommand_FormatText(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
:PROPERTIES:
:ID: text-test
:END:
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::ID:text-test", "--format", "text"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Title:") {
		t.Errorf("text format should contain Title:, got: %s", output)
	}
	if !strings.Contains(output, "Level:") {
		t.Errorf("text format should contain Level:, got: %s", output)
	}
}

func TestGetCommand_NotFound(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{path + "::ID:nonexistent"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent ref")
	}
}

func TestGetCommand_FileNotFound(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	cmd := NewCmdGet(f, nil)
	cmd.SetArgs([]string{"/nonexistent/file.org::ID:test"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGetCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	called := false
	cmd := NewCmdGet(f, func(opts *GetOptions) error {
		called = true
		if opts.Ref == "" {
			t.Error("ref should not be empty")
		}
		return nil
	})
	cmd.SetArgs([]string{"path.org::ID:test"})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}
