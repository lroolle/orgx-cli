package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/ir"
)

func TestParseOrgFile(t *testing.T) {
	content := `#+TITLE: Test Doc

* TODO First Heading :tag1:tag2:
:PROPERTIES:
:ID: test-id-123
:END:
Body content here.

** Subheading
Sub body.

* Second Heading
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if doc.DocType != ir.DocTypeOrg {
		t.Errorf("DocType = %v, want org", doc.DocType)
	}

	if doc.Meta.Title != "Test Doc" {
		t.Errorf("Title = %q, want %q", doc.Meta.Title, "Test Doc")
	}

	if len(doc.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(doc.Nodes))
	}

	h1, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	if h1.Title != "First Heading" {
		t.Errorf("Title = %q, want %q", h1.Title, "First Heading")
	}

	if h1.Todo != "TODO" {
		t.Errorf("Todo = %q, want TODO", h1.Todo)
	}

	if len(h1.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(h1.Tags))
	}

	if h1.Props["ID"] != "test-id-123" {
		t.Errorf("Props[ID] = %q, want test-id-123", h1.Props["ID"])
	}

	expectedRef := path + "::ID:test-id-123"
	if h1.Ref != expectedRef {
		t.Errorf("Ref = %q, want %q", h1.Ref, expectedRef)
	}

	if len(h1.Children) != 1 {
		t.Fatalf("len(Children) = %d, want 1", len(h1.Children))
	}
}

func TestParseMdFile(t *testing.T) {
	content := `---
title: Markdown Test
author: tester
---

# First Heading

Content here.

## Subheading

More content.

# Second Heading
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if doc.DocType != ir.DocTypeMarkdown {
		t.Errorf("DocType = %v, want md", doc.DocType)
	}

	if doc.Meta.Title != "Markdown Test" {
		t.Errorf("Title = %q, want %q", doc.Meta.Title, "Markdown Test")
	}

	if doc.Meta.Frontmatter["author"] != "tester" {
		t.Errorf("Frontmatter[author] = %v, want tester", doc.Meta.Frontmatter["author"])
	}

	if len(doc.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(doc.Nodes))
	}

	h1, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	if h1.Title != "First Heading" {
		t.Errorf("Title = %q, want %q", h1.Title, "First Heading")
	}

	if h1.Level != 1 {
		t.Errorf("Level = %d, want 1", h1.Level)
	}
}

func TestParseUnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for unsupported file type")
	}
}

func TestParseOrgFile_CustomState(t *testing.T) {
	content := `* IDEA Captured thought
:PROPERTIES:
:ID: idea-123
:END:
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	if h.Todo != "IDEA" {
		t.Errorf("Todo = %q, want IDEA", h.Todo)
	}
	if h.Title != "Captured thought" {
		t.Errorf("Title = %q, want 'Captured thought'", h.Title)
	}
}

func TestParseOrgFile_LogbookMultipleEntries(t *testing.T) {
	content := `* DONE Task
:PROPERTIES:
:ID: task-123
:END:
:LOGBOOK:
- State "DONE"        from "STRT"        [2026-01-09 Fri 15:00]
- State "STRT"        from "TODO"        [2026-01-09 Fri 14:00]
- State "TODO"        from "IDEA"        [2026-01-09 Fri 13:00]
:END:
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	if len(h.Logbook) != 3 {
		t.Errorf("len(Logbook) = %d, want 3", len(h.Logbook))
	}

	if len(h.Logbook) >= 1 && h.Logbook[0].NewState != "DONE" {
		t.Errorf("Logbook[0].NewState = %q, want DONE", h.Logbook[0].NewState)
	}
	if len(h.Logbook) >= 2 && h.Logbook[1].NewState != "STRT" {
		t.Errorf("Logbook[1].NewState = %q, want STRT", h.Logbook[1].NewState)
	}
	if len(h.Logbook) >= 3 && h.Logbook[2].NewState != "TODO" {
		t.Errorf("Logbook[2].NewState = %q, want TODO", h.Logbook[2].NewState)
	}
}

func TestParseOrgFile_CustomStateFalsePositive(t *testing.T) {
	// NEXTGEN should NOT be parsed as NEXT state
	// The state keyword must be followed by a space
	content := `* NEXTGEN project
:PROPERTIES:
:ID: nextgen-123
:END:
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := &OrgParser{StateKeywords: []string{"NEXT", "TODO", "DONE"}}
	doc, err := p.Parse(path, []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	// NEXTGEN should remain in title, not be split as NEXT state
	if h.Todo != "" {
		t.Errorf("Todo = %q, want empty (NEXTGEN is not NEXT)", h.Todo)
	}
	if h.Title != "NEXTGEN project" {
		t.Errorf("Title = %q, want 'NEXTGEN project'", h.Title)
	}
}

func TestParseOrgFile_PropertyDrawerAfterPlanningLine(t *testing.T) {
	// When CLOSED: appears before :PROPERTIES:, ID should still be found
	content := `* DONE Completed task
CLOSED: [2026-01-10 Sat 15:00]
:PROPERTIES:
:ID: after-planning-123
:CREATED: [2026-01-09 Fri 10:00]
:END:
Body content.
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	// ID should be found even with CLOSED before PROPERTIES
	if h.Props["ID"] != "after-planning-123" {
		t.Errorf("Props[ID] = %q, want 'after-planning-123'", h.Props["ID"])
	}

	// Ref should use ID, not outline path
	expectedRef := path + "::ID:after-planning-123"
	if h.Ref != expectedRef {
		t.Errorf("Ref = %q, want %q", h.Ref, expectedRef)
	}

	// Closed should be extracted
	if h.Closed == "" {
		t.Error("Closed should be extracted")
	}
}

func TestParseOrgFile_LogbookAsOrgList(t *testing.T) {
	// LOGBOOK formatted as org list items (how go-org actually tokenizes it)
	// This ensures we don't regress if go-org changes internal representation
	content := `* DONE Task with list logbook
:PROPERTIES:
:ID: list-logbook-123
:END:
:LOGBOOK:
- State "DONE"        from "STRT"        [2026-01-10 Sat 16:00]
- State "STRT"        from "TODO"        [2026-01-10 Sat 15:00]
:END:
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	if len(h.Logbook) != 2 {
		t.Fatalf("len(Logbook) = %d, want 2", len(h.Logbook))
	}

	// Verify both entries are parsed correctly
	if h.Logbook[0].NewState != "DONE" || h.Logbook[0].OldState != "STRT" {
		t.Errorf("Logbook[0] = %+v, want DONE<-STRT", h.Logbook[0])
	}
	if h.Logbook[1].NewState != "STRT" || h.Logbook[1].OldState != "TODO" {
		t.Errorf("Logbook[1] = %+v, want STRT<-TODO", h.Logbook[1])
	}
}

func TestParseOrgFile_MultiKeywordPlanningLine(t *testing.T) {
	// Single planning line with multiple keywords - all should be parsed
	content := `* TODO Task with multiple planning keywords
SCHEDULED: <2026-01-15 Thu> DEADLINE: <2026-01-20 Tue>
:PROPERTIES:
:ID: multi-planning-123
:END:
Body content.
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	// Both SCHEDULED and DEADLINE should be extracted
	if h.Scheduled == "" {
		t.Error("Scheduled should be extracted from multi-keyword line")
	}
	if h.Deadline == "" {
		t.Error("Deadline should be extracted from multi-keyword line")
	}
}

func TestParseOrgFile_AllThreePlanningKeywords(t *testing.T) {
	// Single line with CLOSED + SCHEDULED + DEADLINE
	content := `* DONE Completed task with all planning keywords
CLOSED: [2026-01-10 Sat 15:00] SCHEDULED: <2026-01-08 Thu> DEADLINE: <2026-01-09 Fri>
:PROPERTIES:
:ID: all-three-123
:END:
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	if h.Closed == "" {
		t.Error("Closed should be extracted")
	}
	if h.Scheduled == "" {
		t.Error("Scheduled should be extracted")
	}
	if h.Deadline == "" {
		t.Error("Deadline should be extracted")
	}
}

func TestParseOrgFile_DeadlineInBodyNotMatched(t *testing.T) {
	// DEADLINE: appearing in body text should NOT be parsed as planning
	// Planning must come immediately after headline, before drawers/body
	content := `* TODO Task with deadline mentioned in body
:PROPERTIES:
:ID: body-deadline-123
:END:

This task has no actual deadline, but mentions DEADLINE: <2026-01-20 Tue> in the text.
Also mentions SCHEDULED: <2026-01-15 Thu> and CLOSED: [2026-01-10 Sat] in prose.
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatal("first node is not a Heading")
	}

	// None of these should be set - they're in body, not planning line
	if h.Deadline != "" {
		t.Errorf("Deadline = %q, want empty (DEADLINE in body should not match)", h.Deadline)
	}
	if h.Scheduled != "" {
		t.Errorf("Scheduled = %q, want empty (SCHEDULED in body should not match)", h.Scheduled)
	}
	if h.Closed != "" {
		t.Errorf("Closed = %q, want empty (CLOSED in body should not match)", h.Closed)
	}
}

func TestHeadlineTitleLinksAreExtracted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daily.org")
	content := "* 14:30 worked on [[id:abc-123][SRP notes]]  :@claude:\nBody mentions [[id:def-456][other]].\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h, ok := doc.Nodes[0].(*ir.Heading)
	if !ok {
		t.Fatalf("node = %T", doc.Nodes[0])
	}
	if len(h.Links) != 2 {
		t.Fatalf("want title+body links, got %d: %+v", len(h.Links), h.Links)
	}
	// Title link first — it is the entry's primary reference.
	if h.Links[0].Target != "id:abc-123" && h.Links[0].Target != "abc-123" {
		t.Fatalf("title link = %+v", h.Links[0])
	}
}
