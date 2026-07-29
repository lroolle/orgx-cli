package daily

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/roam"
)

var clock = time.Date(2026, 7, 29, 14, 30, 0, 0, time.UTC)

func opts(t *testing.T, dir, text, as string) *DailyOptions {
	t.Helper()
	io, _, _, _ := iostreams.Test()
	return &DailyOptions{
		IO: io, DailiesDir: dir, Text: text, As: as, Now: clock, Confirmed: true,
	}
}

func TestFirstEntryCreatesARealNode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "daily")
	if err := dailyRun(opts(t, dir, "reviewed the SRP notes", "")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "2026-07-29.org")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{":PROPERTIES:", ":ID:", "#+title: 2026-07-29", "* 14:30 reviewed the SRP notes"} {
		if !strings.Contains(content, want) {
			t.Fatalf("daily missing %q:\n%s", want, content)
		}
	}
	// The daily must be scannable as a node.
	meta, err := roam.ReadMeta(path)
	if err != nil || meta.ID == "" || meta.Title != "2026-07-29" {
		t.Fatalf("daily is not a node: %+v %v", meta, err)
	}
}

func TestAgentEntriesCarryTheAuthorTag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "daily")
	if err := dailyRun(opts(t, dir, "human note", "")); err != nil {
		t.Fatal(err)
	}
	if err := dailyRun(opts(t, dir, "reserved 3 aliases", "claude")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "2026-07-29.org"))
	content := string(raw)
	if !strings.Contains(content, "* 14:30 reserved 3 aliases  :@claude:") {
		t.Fatalf("agent attribution missing:\n%s", content)
	}
	// Appending must not duplicate the preamble.
	if strings.Count(content, "#+title:") != 1 {
		t.Fatalf("preamble duplicated:\n%s", content)
	}
}

func TestShowWithoutTextDoesNotWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "daily")
	o := opts(t, dir, "", "")
	if err := dailyRun(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-07-29.org")); !os.IsNotExist(err) {
		t.Fatal("show created a file — reads must never write")
	}
}

func TestAppendRequiresConfirmationOffTTY(t *testing.T) {
	o := opts(t, t.TempDir(), "text", "")
	o.Confirmed = false
	if err := dailyRun(o); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v, want --yes requirement", err)
	}
}
