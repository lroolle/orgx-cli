package backlinks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestBacklinksCommand_Basic(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.org"), []byte("* A\n[[file:target.org][Link to target]]\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.org"), []byte("* B\n[[file:target.org][Another link]]\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "target.org"), []byte("* Target\n"), 0644)

	cmd := NewCmdBacklinks(f, nil)
	cmd.SetArgs([]string{"target.org", "--in", tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "2 backlinks") {
		t.Errorf("should find 2 backlinks, got: %s", output)
	}
}

func TestBacklinksCommand_JSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "source.org"), []byte("* Source\n[[file:target.org][Link]]\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "target.org"), []byte("* Target\n"), 0644)

	cmd := NewCmdBacklinks(f, nil)
	cmd.SetArgs([]string{"target.org", "--in", tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var backlinks []BacklinkOutput
	if err := json.Unmarshal(stdout.Bytes(), &backlinks); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(backlinks) != 1 {
		t.Fatalf("expected 1 backlink, got %d", len(backlinks))
	}
	if backlinks[0].SourceTitle != "Source" {
		t.Errorf("source title = %q, want Source", backlinks[0].SourceTitle)
	}
}

func TestBacklinksCommand_NoBacklinks(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "orphan.org"), []byte("* Orphan\n"), 0644)

	cmd := NewCmdBacklinks(f, nil)
	cmd.SetArgs([]string{"orphan.org", "--in", tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "No backlinks") {
		t.Errorf("should say no backlinks, got: %s", stdout.String())
	}
}

func TestBacklinksCommand_IDLinks(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "source.org"), []byte("* Source\n[[id:abc-123][ID Link]]\n"), 0644)

	cmd := NewCmdBacklinks(f, nil)
	cmd.SetArgs([]string{"abc-123", "--in", tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var backlinks []BacklinkOutput
	json.Unmarshal(stdout.Bytes(), &backlinks)

	if len(backlinks) != 1 {
		t.Errorf("expected 1 ID backlink, got %d", len(backlinks))
	}
}

func TestBacklinksCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	called := false
	cmd := NewCmdBacklinks(f, func(opts *BacklinksOptions) error {
		called = true
		if opts.Target != "test.org" {
			t.Errorf("target = %q, want test.org", opts.Target)
		}
		return nil
	})
	cmd.SetArgs([]string{"test.org", "--in", "."})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}
