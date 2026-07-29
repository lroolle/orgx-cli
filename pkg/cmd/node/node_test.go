package node

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/roam"
)

func testIO() (*iostreams.IOStreams, *bytes.Buffer) {
	io, _, out, _ := iostreams.Test()
	return io, out
}

func TestNewCreatesAnOrgRoamCompatibleNode(t *testing.T) {
	root := t.TempDir()
	io, _ := testIO()
	opts := &NewOptions{
		IO: io, Root: root, Title: "SRP protocol notes",
		Tags: []string{"auth"}, As: "claude", Confirmed: true,
	}
	if err := newRun(opts); err != nil {
		t.Fatal(err)
	}

	nodes, _, err := roam.Scan(root)
	if err != nil || len(nodes) != 1 {
		t.Fatalf("scan: %v %+v", err, nodes)
	}
	n := nodes[0]
	if n.Title != "SRP protocol notes" || n.ID == "" || n.Author != "claude" {
		t.Fatalf("node = %+v", n)
	}
	// Author rides both the property and an @tag, so find --tag works.
	if len(n.Tags) != 2 || n.Tags[0] != "auth" || n.Tags[1] != "@claude" {
		t.Fatalf("tags = %v", n.Tags)
	}
	base := filepath.Base(n.Path)
	if !strings.HasSuffix(base, "-srp_protocol_notes.org") || len(base) < 20 {
		t.Fatalf("filename not org-roam shaped: %s", base)
	}
	raw, _ := os.ReadFile(n.Path)
	for _, want := range []string{":PROPERTIES:", ":ID:", ":CREATED:", "#+title: SRP protocol notes", "#+filetags: :auth:@claude:"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("file missing %q:\n%s", want, raw)
		}
	}
}

func TestNewRefusesWithoutConfirmationWhenNotATTY(t *testing.T) {
	io, _ := testIO()
	opts := &NewOptions{IO: io, Root: t.TempDir(), Title: "x"}
	if err := newRun(opts); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v, want --yes requirement", err)
	}
}

func TestListFiltersAndReportsSkipped(t *testing.T) {
	root := t.TempDir()
	io, _ := testIO()
	for _, n := range []struct{ title, tag string }{
		{"Auth notes", "auth"}, {"Cooking", "home"},
	} {
		if err := newRun(&NewOptions{IO: io, Root: root, Title: n.title, Tags: []string{n.tag}, Confirmed: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "loose.org"), []byte("#+title: no id\n"), 0644); err != nil {
		t.Fatal(err)
	}

	io2, out := testIO()
	if err := listRun(&ListOptions{IO: io2, Root: root, Tag: "auth"}); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "Auth notes") || strings.Contains(text, "Cooking") {
		t.Fatalf("tag filter broken:\n%s", text)
	}
	if !strings.Contains(text, "without a file-level :ID:") {
		t.Fatalf("skipped files hidden:\n%s", text)
	}
}
