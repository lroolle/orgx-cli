package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/ir"
)

func TestMdParser_BasicHeading(t *testing.T) {
	content := `# Simple Heading

Body text here.
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}

	h := doc.Nodes[0].(*ir.Heading)
	if h.Title != "Simple Heading" {
		t.Errorf("title = %q, want %q", h.Title, "Simple Heading")
	}
	if h.Level != 1 {
		t.Errorf("level = %d, want 1", h.Level)
	}
}

func TestMdParser_HeadingLevels(t *testing.T) {
	tests := []struct {
		name    string
		content string
		level   int
	}{
		{"h1", "# Heading\n", 1},
		{"h2", "## Heading\n", 2},
		{"h3", "### Heading\n", 3},
		{"h4", "#### Heading\n", 4},
		{"h5", "##### Heading\n", 5},
		{"h6", "###### Heading\n", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseMdString(t, tt.content)
			if len(doc.Nodes) == 0 {
				t.Fatal("no headings found")
			}
			h := doc.Nodes[0].(*ir.Heading)
			if h.Level != tt.level {
				t.Errorf("level = %d, want %d", h.Level, tt.level)
			}
		})
	}
}

func TestMdParser_Frontmatter(t *testing.T) {
	content := `---
title: My Document
author: Test Author
date: 2024-01-01
tags:
  - tag1
  - tag2
---

# Heading
`
	doc := parseMdString(t, content)

	if doc.Meta.Title != "My Document" {
		t.Errorf("title = %q, want %q", doc.Meta.Title, "My Document")
	}

	if doc.Meta.Frontmatter == nil {
		t.Fatal("frontmatter is nil")
	}

	if doc.Meta.Frontmatter["author"] != "Test Author" {
		t.Errorf("author = %v, want %q", doc.Meta.Frontmatter["author"], "Test Author")
	}
}

func TestMdParser_NestedHeadings(t *testing.T) {
	content := `# Level 1

## Level 2

### Level 3

## Another Level 2

# Another Level 1
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 2 {
		t.Fatalf("expected 2 top-level headings, got %d", len(doc.Nodes))
	}

	h1 := doc.Nodes[0].(*ir.Heading)
	if len(h1.Children) != 2 {
		t.Errorf("first h1 should have 2 children, got %d", len(h1.Children))
	}
}

func TestMdParser_HashRef(t *testing.T) {
	content := `# My Heading
`
	doc := parseMdString(t, content)
	h := doc.Nodes[0].(*ir.Heading)

	if !strings.Contains(h.Ref, "::H:") {
		t.Errorf("ref = %q, should contain ::H: for hash ref", h.Ref)
	}
}

func TestMdParser_CodeBlocks(t *testing.T) {
	content := "# Heading\n\n```python\ndef hello():\n    print('hello')\n```\n"
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestMdParser_InlineCode(t *testing.T) {
	content := "# Heading with `inline code`\n"
	doc := parseMdString(t, content)

	h := doc.Nodes[0].(*ir.Heading)
	if !strings.Contains(h.Title, "inline code") {
		t.Errorf("title should contain inline code text, got %q", h.Title)
	}
}

func TestMdParser_Links(t *testing.T) {
	content := `# Heading with [link](https://example.com)

Body with [another link](file.md).
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestMdParser_Lists(t *testing.T) {
	content := `# Heading

- Item 1
- Item 2
  - Nested
- Item 3

1. Numbered 1
2. Numbered 2
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestMdParser_Blockquote(t *testing.T) {
	content := `# Heading

> This is a quote
> spanning multiple lines
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestMdParser_EmptyFile(t *testing.T) {
	content := ``
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 0 {
		t.Errorf("expected 0 headings for empty file, got %d", len(doc.Nodes))
	}
}

func TestMdParser_OnlyFrontmatter(t *testing.T) {
	content := `---
title: Only Frontmatter
---
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 0 {
		t.Errorf("expected 0 headings, got %d", len(doc.Nodes))
	}
	if doc.Meta.Title != "Only Frontmatter" {
		t.Errorf("title = %q, want %q", doc.Meta.Title, "Only Frontmatter")
	}
}

func TestMdParser_SHA256(t *testing.T) {
	content := `# Heading

Body.
`
	doc := parseMdString(t, content)

	if doc.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}
	if len(doc.SHA256) != 64 {
		t.Errorf("SHA256 should be 64 chars, got %d", len(doc.SHA256))
	}
}

func TestMdParser_DocType(t *testing.T) {
	content := `# Heading
`
	doc := parseMdString(t, content)

	if doc.DocType != ir.DocTypeMarkdown {
		t.Errorf("DocType = %v, want %v", doc.DocType, ir.DocTypeMarkdown)
	}
}

func TestMdParser_Tables(t *testing.T) {
	content := `# Table Example

| Name | Age |
|------|-----|
| John | 30  |
| Jane | 25  |
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestMdParser_SpecialCharsInTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"colon", "# Heading: with colon\n", "Heading: with colon"},
		{"parens", "# Heading (with parens)\n", "Heading (with parens)"},
		{"brackets", "# Heading [with brackets]\n", "Heading [with brackets]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseMdString(t, tt.content)
			if len(doc.Nodes) == 0 {
				t.Fatal("no headings found")
			}
			h := doc.Nodes[0].(*ir.Heading)
			if h.Title != tt.want {
				t.Errorf("title = %q, want %q", h.Title, tt.want)
			}
		})
	}
}

func TestMdParser_HorizontalRule(t *testing.T) {
	content := `# First

---

# Second
`
	doc := parseMdString(t, content)

	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 headings, got %d", len(doc.Nodes))
	}
}

func TestMdParser_EmphasisInTitle(t *testing.T) {
	content := `# Heading with **bold** and *italic*
`
	doc := parseMdString(t, content)

	h := doc.Nodes[0].(*ir.Heading)
	if !strings.Contains(h.Title, "bold") {
		t.Errorf("title should contain bold text, got %q", h.Title)
	}
}

func parseMdString(t *testing.T, content string) *ir.Document {
	t.Helper()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	return doc
}
