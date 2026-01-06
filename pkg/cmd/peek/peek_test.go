package peek

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestPeekCommand_Basic(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `#+TITLE: Test
* TODO Heading 1 :tag1:
** Subheading
* Heading 2
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdPeek(f, nil)
	cmd.SetArgs([]string{path})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "3 headings") {
		t.Errorf("output should contain heading count, got: %s", output)
	}
	if !strings.Contains(output, "TODO") {
		t.Errorf("output should contain TODO state, got: %s", output)
	}
	if !strings.Contains(output, ":tag1:") {
		t.Errorf("output should contain tag, got: %s", output)
	}
}

func TestPeekCommand_JSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading 1
** Subheading
* Heading 2
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdPeek(f, nil)
	cmd.SetArgs([]string{path, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var output PeekOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if output.Headings != 3 {
		t.Errorf("heading count = %d, want 3", output.Headings)
	}
	if len(output.Entries) != 3 {
		t.Errorf("entries = %d, want 3", len(output.Entries))
	}
}

func TestPeekCommand_MaxDepth(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Level 1
** Level 2
*** Level 3
**** Level 4
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdPeek(f, nil)
	cmd.SetArgs([]string{path, "--max-depth", "2", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var output PeekOutput
	json.Unmarshal(stdout.Bytes(), &output)

	for _, e := range output.Entries {
		if e.Level > 2 {
			t.Errorf("found heading at level %d, should be max 2", e.Level)
		}
	}
}

func TestPeekCommand_FileNotFound(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	cmd := NewCmdPeek(f, nil)
	cmd.SetArgs([]string{"/nonexistent/file.org"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestPeekCommand_Markdown(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	content := `# Heading 1

## Subheading

# Heading 2
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdPeek(f, nil)
	cmd.SetArgs([]string{path, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var output PeekOutput
	json.Unmarshal(stdout.Bytes(), &output)

	if output.Headings != 3 {
		t.Errorf("heading count = %d, want 3", output.Headings)
	}
}

func TestPeekCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	os.WriteFile(path, []byte("* Heading\n"), 0644)

	called := false
	cmd := NewCmdPeek(f, func(opts *PeekOptions) error {
		called = true
		if opts.Path != path {
			t.Errorf("path = %q, want %q", opts.Path, path)
		}
		return nil
	})
	cmd.SetArgs([]string{path})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}

func TestExtractRefShort(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"path.org::ID:abc123", "::ID:abc123"},
		{"path.org::/Outline", "::/Outline"},
		{"path.md::H:1a2b3c", "::H:1a2b3c"},
		{"path.org", ""},
	}

	for _, tt := range tests {
		got := extractRefShort(tt.ref)
		if got != tt.want {
			t.Errorf("extractRefShort(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}
