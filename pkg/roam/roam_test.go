package roam

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/config"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReadMetaExtractsFileLevelIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "srp.org")
	write(t, path, `:PROPERTIES:
:ID:       abc-123
:ORGX_AUTHOR: claude
:END:
#+title: SRP protocol notes
#+filetags: :auth:apple:@claude:

* First heading
:PROPERTIES:
:ID: heading-id-must-not-win
:END:
`)
	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ID != "abc-123" || meta.Title != "SRP protocol notes" || meta.Author != "claude" {
		t.Fatalf("meta = %+v", meta)
	}
	if len(meta.Tags) != 3 || meta.Tags[2] != "@claude" {
		t.Fatalf("tags = %v", meta.Tags)
	}
}

func TestReadMetaStopsAtFirstHeading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.org")
	write(t, path, "* Just a heading\n#+title: not a file title\n")
	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ID != "" || meta.Title != "" {
		t.Fatalf("preamble leaked past first heading: %+v", meta)
	}
}

func TestScanCountsNonNodesInsteadOfHidingThem(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.org"), ":PROPERTIES:\n:ID: id-a\n:END:\n#+title: A\n")
	write(t, filepath.Join(root, "sub", "b.org"), ":PROPERTIES:\n:ID: id-b\n:END:\n#+title: B\n")
	write(t, filepath.Join(root, "no-id.org"), "#+title: not a node\n")
	write(t, filepath.Join(root, "readme.md"), "# ignored\n")

	nodes, skipped, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || skipped != 1 {
		t.Fatalf("nodes=%d skipped=%d", len(nodes), skipped)
	}
	if nodes[0].Title != "A" || nodes[1].Title != "B" {
		t.Fatalf("order/titles = %+v", nodes)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"SRP protocol notes":   "srp_protocol_notes",
		"Retro 2026-07":        "retro_2026_07",
		"  weird -- spacing  ": "weird_spacing",
		"聊天记录":                 "untitled",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Fatalf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveRootPrecedence(t *testing.T) {
	cfg := &config.Config{
		Version:          1,
		DefaultWorkspace: "main",
		Workspaces:       map[string]config.Workspace{"main": {Root: "/ws/root"}},
	}
	if root, err := ResolveRoot(cfg, "", "/override"); err != nil || root != "/override" {
		t.Fatalf("override: %q %v", root, err)
	}
	if root, err := ResolveRoot(cfg, "", ""); err != nil || root != "/ws/root" {
		t.Fatalf("default ws: %q %v", root, err)
	}
	empty := &config.Config{Version: 1, Workspaces: map[string]config.Workspace{}}
	if _, err := ResolveRoot(empty, "", ""); err == nil {
		t.Fatal("want fix-bearing error with no workspace")
	}
}

func TestDailiesDirDefaultsAndConfig(t *testing.T) {
	cfg := &config.Config{
		Version:          1,
		DefaultWorkspace: "main",
		Workspaces:       map[string]config.Workspace{"main": {Root: "/ws", Dailies: "journal"}},
	}
	if got := DailiesDir(cfg, "", "/ws"); got != filepath.Join("/ws", "journal") {
		t.Fatalf("configured dailies = %q", got)
	}
	// No workspace config: the vault layout decides — journals by
	// default, org-roam's daily/ when that convention is present.
	plain := &config.Config{Version: 1, Workspaces: map[string]config.Workspace{}}
	fresh := t.TempDir()
	if got := DailiesDir(plain, "", fresh); got != filepath.Join(fresh, "journals") {
		t.Fatalf("default dailies = %q", got)
	}
	orgroam := t.TempDir()
	if err := os.MkdirAll(filepath.Join(orgroam, "daily"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := DailiesDir(plain, "", orgroam); got != filepath.Join(orgroam, "daily") {
		t.Fatalf("org-roam dailies = %q", got)
	}
}
