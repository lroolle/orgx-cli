package ls

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func TestLsCommand_Basic(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.org"), []byte("* Heading 1\n* Heading 2\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.org"), []byte("* Single\n"), 0644)

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "a.org") {
		t.Errorf("should list a.org, got: %s", output)
	}
	if !strings.Contains(output, "b.org") {
		t.Errorf("should list b.org, got: %s", output)
	}
}

func TestLsCommand_JSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.org"), []byte("#+TITLE: Test\n* H1\n* H2\n"), 0644)

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir, "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var files []FileInfo
	if err := json.Unmarshal(stdout.Bytes(), &files); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Headings != 2 {
		t.Errorf("headings = %d, want 2", files[0].Headings)
	}
	if files[0].Title != "Test" {
		t.Errorf("title = %q, want Test", files[0].Title)
	}
}

func TestLsCommand_Limit(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(tmpDir, string(rune('a'+i))+".org"), []byte("* H\n"), 0644)
	}

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir, "--limit", "3", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var files []FileInfo
	json.Unmarshal(stdout.Bytes(), &files)

	if len(files) != 3 {
		t.Errorf("expected 3 files with limit, got %d", len(files))
	}
}

func TestLsCommand_SortByLines(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "small.org"), []byte("* H\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "big.org"), []byte("* H1\n* H2\n* H3\n* H4\n* H5\n"), 0644)

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir, "--sort", "lines", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var files []FileInfo
	json.Unmarshal(stdout.Bytes(), &files)

	if len(files) < 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Name != "big.org" {
		t.Errorf("first file should be big.org (most lines), got %s", files[0].Name)
	}
}

func TestLsCommand_Recursive(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "root.org"), []byte("* R\n"), 0644)
	os.WriteFile(filepath.Join(subDir, "child.org"), []byte("* C\n"), 0644)

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir, "--recursive", "--json"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var files []FileInfo
	json.Unmarshal(stdout.Bytes(), &files)

	if len(files) != 2 {
		t.Errorf("expected 2 files with recursive, got %d", len(files))
	}
}

func TestLsCommand_NoFiles(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "No org/markdown") {
		t.Errorf("should say no files, got: %s", stdout.String())
	}
}

func TestLsCommand_TruncationHint(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	tmpDir := t.TempDir()
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(tmpDir, string(rune('a'+i))+".org"), []byte("* H\n"), 0644)
	}

	cmd := NewCmdLs(f, nil)
	cmd.SetArgs([]string{tmpDir, "--limit", "3"})
	cmd.SetOut(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "showing 3/5") {
		t.Errorf("should show truncation hint, got: %s", stdout.String())
	}
}

func TestLsCommand_RunF(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios}

	called := false
	cmd := NewCmdLs(f, func(opts *LsOptions) error {
		called = true
		if !opts.Recursive {
			t.Error("recursive should be true")
		}
		return nil
	})
	cmd.SetArgs([]string{".", "--recursive"})
	cmd.SetOut(stdout)
	cmd.Execute()

	if !called {
		t.Error("runF was not called")
	}
}
