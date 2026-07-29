package agentcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/roam"
)

func testVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := roam.InitVault(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pages", "flashcards.org"),
		[]byte(":PROPERTIES:\n:ID: id-cards\n:END:\n#+title: flashcards\n\n- the user prefers surgical diffs\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "journals", "2026-07-28.org"),
		[]byte(":PROPERTIES:\n:ID: id-day\n:END:\n#+title: 2026-07-28\n\n* 10:00 shipped the roam layer  :@claude:\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestBriefCarriesVaultAndRules(t *testing.T) {
	root := testVault(t)
	brief, err := buildBrief(root, "claude")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<orgx-vault>",
		root,
		"one node per day",
		"the user prefers surgical diffs", // flashcards tail injected
		"shipped the roam layer",          // recent journal injected
		"--as claude --yes",               // author substituted into the rule
		"[[id:<uuid>]]",                   // link discipline
		"orgx id check",                   // verify-after-freehand
		"</orgx-vault>",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief missing %q:\n%s", want, brief)
		}
	}
}

func TestBriefSurvivesEmptyVault(t *testing.T) {
	root := t.TempDir()
	brief, err := buildBrief(root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(brief, "0 nodes") {
		t.Fatalf("empty vault should still brief:\n%s", brief)
	}
	if strings.Contains(brief, "Standing context") {
		t.Fatal("no flashcards file must mean no standing-context section")
	}
}

func TestTailBytesKeepsWholeLines(t *testing.T) {
	s := "first line\nsecond line\nthird line"
	got := tailBytes(s, 15)
	if got != "third line" {
		t.Fatalf("want trailing whole lines, got %q", got)
	}
	if tailBytes(s, 1000) != s {
		t.Fatal("under budget must be untouched")
	}
}

func TestInvocationShapes(t *testing.T) {
	brief := "<orgx-vault>...</orgx-vault>"

	got := invocation("claude", brief, "garden the vault")
	want := []string{"claude", "--append-system-prompt", brief, "-p", "garden the vault"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("claude one-shot wrong: %v", got)
	}

	if got := invocation("claude", brief, ""); len(got) != 3 || got[len(got)-1] != brief {
		t.Fatalf("claude interactive wrong: %v", got)
	}

	got = invocation("codex", brief, "fix broken links")
	if got[0] != "codex" || got[1] != "exec" || !strings.Contains(got[2], "Task: fix broken links") ||
		!strings.Contains(got[2], "<orgx-vault>") {
		t.Fatalf("codex one-shot must carry brief in the prompt: %v", got)
	}
}

func TestDetectToolPrefersClaudeAndErrsWithFix(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"claude", "codex"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)

	if tool, err := detectTool(""); err != nil || tool != "claude" {
		t.Fatalf("want claude first, got %q err %v", tool, err)
	}
	if tool, err := detectTool("codex"); err != nil || tool != "codex" {
		t.Fatalf("explicit codex, got %q err %v", tool, err)
	}

	t.Setenv("PATH", t.TempDir()) // nothing installed
	if _, err := detectTool(""); err == nil || !strings.Contains(err.Error(), "--brief") {
		t.Fatalf("empty PATH must error and point at --brief, got %v", err)
	}
	if _, err := detectTool("aider"); err == nil {
		t.Fatal("unknown tool must error")
	}
}

func TestStripFileHeadDropsDrawerAndKeywords(t *testing.T) {
	in := ":PROPERTIES:\n:ID: x\n:END:\n#+title: flashcards\n\n- durable fact one\n- durable fact two\n"
	if got := stripFileHead(in); got != "- durable fact one\n- durable fact two" {
		t.Fatalf("head not stripped: %q", got)
	}
	if got := stripFileHead(":PROPERTIES:\n:ID: x\n:END:\n#+title: empty\n\n"); got != "" {
		t.Fatalf("head-only file should strip to nothing, got %q", got)
	}
}
