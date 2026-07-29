package find

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectorySearchRecursesIntoSubdirectories(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"top.org", "daily/2026-07-29.org", "topics/deep/auth.org", "notes.md"} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("* heading\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := findFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Fatalf("want 4 files across subdirs, got %d: %v", len(files), files)
	}
}

func TestGlobPatternsStillWork(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"a.org", "b.org", "sub/c.org"} {
		full := filepath.Join(root, p)
		_ = os.MkdirAll(filepath.Dir(full), 0755)
		_ = os.WriteFile(full, []byte("* h\n"), 0644)
	}
	files, err := findFiles(filepath.Join(root, "*.org"))
	if err != nil {
		t.Fatal(err)
	}
	// A glob is an explicit scope: exactly what it matches, no walk.
	if len(files) != 2 {
		t.Fatalf("glob must stay non-recursive, got %v", files)
	}
}
