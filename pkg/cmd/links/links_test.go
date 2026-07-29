package links

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestLinksCommand_Basic(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
Check out [[https://example.com][Example]] and [[file:other.org][Other File]].
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdLinks(f, nil)
	cmd.SetArgs([]string{path})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "https://example.com") {
		t.Errorf("output should contain http link, got: %s", output)
	}
	if !strings.Contains(output, "other.org") {
		t.Errorf("output should contain file link, got: %s", output)
	}
}

func TestLinksCommand_JSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
[[https://example.com][Example Link]]
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdLinks(f, nil)
	cmd.SetArgs([]string{path, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var env struct {
		Items []LinkOutput `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	links := env.Items

	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Kind != "http" {
		t.Errorf("kind = %q, want http", links[0].Kind)
	}
}

func TestLinksCommand_KindFilter(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	content := `* Heading
[[https://example.com][HTTP]]
[[file:other.org][File]]
`
	os.WriteFile(path, []byte(content), 0644)

	cmd := NewCmdLinks(f, nil)
	cmd.SetArgs([]string{path, "--kind", "file", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var env struct {
		Items []LinkOutput `json:"items"`
	}
	json.Unmarshal(stdout.Bytes(), &env)
	links := env.Items

	if len(links) != 1 {
		t.Errorf("expected 1 file link, got %d", len(links))
	}
}

func TestLinksCommand_Directory(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.org"), []byte("* A\n[[file:b.org][B]]\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.org"), []byte("* B\n[[file:a.org][A]]\n"), 0644)

	cmd := NewCmdLinks(f, nil)
	cmd.SetArgs([]string{"--in", tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var env struct {
		Items []LinkOutput `json:"items"`
	}
	json.Unmarshal(stdout.Bytes(), &env)
	links := env.Items

	if len(links) != 2 {
		t.Errorf("expected 2 links from 2 files, got %d", len(links))
	}
}

func TestLinksCommand_NoLinks(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	os.WriteFile(path, []byte("* Heading\nNo links here.\n"), 0644)

	cmd := NewCmdLinks(f, nil)
	cmd.SetArgs([]string{path})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "No links") {
		t.Errorf("should say no links, got: %s", stdout.String())
	}
}

func TestLinksCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	called := false
	cmd := NewCmdLinks(f, func(opts *LinksOptions) error {
		called = true
		if opts.Kind != "file" {
			t.Errorf("kind = %q, want file", opts.Kind)
		}
		return nil
	})
	cmd.SetArgs([]string{"test.org", "--kind", "file"})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}
