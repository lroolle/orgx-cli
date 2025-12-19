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
