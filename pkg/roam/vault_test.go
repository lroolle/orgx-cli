package roam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitScaffoldsTheDefaultLayout(t *testing.T) {
	root := t.TempDir()
	report, err := InitVault(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{".orgx", "journals", "pages", "whiteboards", "assets"} {
		if !dirExists(filepath.Join(root, dir)) {
			t.Fatalf("missing %s/", dir)
		}
	}
	// Seed pages are real nodes.
	for _, page := range []string{"contents", "flashcards"} {
		meta, err := ReadMeta(filepath.Join(root, "pages", page+".org"))
		if err != nil || meta.ID == "" || meta.Title != page {
			t.Fatalf("%s seed is not a node: %+v %v", page, meta, err)
		}
	}
	if len(report.Created) == 0 || len(report.Kept) != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestInitIsIdempotentAndNeverTouchesExistingFiles(t *testing.T) {
	root := t.TempDir()
	if _, err := InitVault(root); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(root, "pages", "flashcards.org")
	if err := os.WriteFile(custom, []byte("my edits\n"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := InitVault(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(custom)
	if string(raw) != "my edits\n" {
		t.Fatal("init overwrote an existing file")
	}
	if len(report.Created) != 0 {
		t.Fatalf("second init created %v", report.Created)
	}
}

func TestFindVaultWalksUpLikeGit(t *testing.T) {
	root := t.TempDir()
	if _, err := InitVault(root); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "pages", "sub", "deeper")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	found, ok := FindVault(deep)
	if !ok || found != root {
		t.Fatalf("FindVault = %q, %v", found, ok)
	}
	if _, ok := FindVault(t.TempDir()); ok {
		t.Fatal("found a vault where none exists")
	}
}

func TestLayoutDetectsOrgRoamDailies(t *testing.T) {
	// A plain org-roam vault: flat nodes, daily/ journals, no marker.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "daily"), 0755); err != nil {
		t.Fatal(err)
	}
	layout := LoadLayout(root)
	if layout.Journals != "daily" {
		t.Fatalf("journals = %q, want org-roam's daily", layout.Journals)
	}
	// A fresh directory gets the vault defaults.
	fresh := LoadLayout(t.TempDir())
	if fresh.Journals != "journals" || fresh.Pages != "pages" {
		t.Fatalf("defaults = %+v", fresh)
	}
}

func TestLayoutHonorsVaultConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, MarkerDir), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := "layout:\n  journals: diary\n  pages: notes\n"
	if err := os.WriteFile(filepath.Join(root, MarkerDir, "config.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	layout := LoadLayout(root)
	if layout.Journals != "diary" || layout.Pages != "notes" {
		t.Fatalf("layout = %+v", layout)
	}
}
