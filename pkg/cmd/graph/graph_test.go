package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func node(t *testing.T, root, rel, id, title, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := ":PROPERTIES:\n:ID: " + id + "\n:END:\n#+title: " + title + "\n" + body
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGraphDerivesEdgesAndBrokenLinks(t *testing.T) {
	root := t.TempDir()
	node(t, root, "pages/a.org", "id-a", "A",
		"* links out\nSee [[id:id-b][B]] and [[id:ghost][gone]].\n")
	node(t, root, "journals/2026-07-29.org", "id-day", "2026-07-29",
		"* 10:00 touched [[id:id-a][A]]  :@claude:\n")
	node(t, root, "pages/b.org", "id-b", "B", "")

	io, _, out, _ := iostreams.Test()
	if err := graphRun(&GraphOptions{IO: io, Root: root}); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "3 nodes, 2 edges, 1 broken links") {
		t.Fatalf("summary wrong:\n%s", text)
	}
	if !strings.Contains(text, "id-a -> ghost") {
		t.Fatalf("broken link not reported:\n%s", text)
	}
}

func TestGraphDeduplicatesRepeatedEdges(t *testing.T) {
	root := t.TempDir()
	node(t, root, "a.org", "id-a", "A",
		"* one [[id:id-b][B]]\n* two [[id:id-b][B again]]\n")
	node(t, root, "b.org", "id-b", "B", "")

	io, _, out, _ := iostreams.Test()
	if err := graphRun(&GraphOptions{IO: io, Root: root}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "2 nodes, 1 edges") {
		t.Fatalf("edges not deduplicated:\n%s", out.String())
	}
}
