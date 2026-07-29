package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/ir"
)

func TestOrgParser_BasicHeading(t *testing.T) {
	content := `* Simple Heading
Body text here.
`
	doc := parseOrgString(t, content)

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

func TestOrgParser_TODOStates(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantTodo string
	}{
		{"TODO", "* TODO Task\n", "TODO"},
		{"DONE", "* DONE Completed\n", "DONE"},
		{"No state", "* Just a heading\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseOrgString(t, tt.content)
			h := doc.Nodes[0].(*ir.Heading)
			if h.Todo != tt.wantTodo {
				t.Errorf("todo = %q, want %q", h.Todo, tt.wantTodo)
			}
		})
	}
}

func TestOrgParser_Tags(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantTags []string
	}{
		{"single tag", "* Heading :tag1:\n", []string{"tag1"}},
		{"multiple tags", "* Heading :tag1:tag2:tag3:\n", []string{"tag1", "tag2", "tag3"}},
		{"no tags", "* Heading\n", nil},
		{"with TODO", "* TODO Task :work:\n", []string{"work"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseOrgString(t, tt.content)
			h := doc.Nodes[0].(*ir.Heading)
			if len(h.Tags) != len(tt.wantTags) {
				t.Errorf("tags = %v, want %v", h.Tags, tt.wantTags)
				return
			}
			for i, tag := range tt.wantTags {
				if h.Tags[i] != tag {
					t.Errorf("tag[%d] = %q, want %q", i, h.Tags[i], tag)
				}
			}
		})
	}
}

func TestOrgParser_Properties(t *testing.T) {
	content := `* Heading
:PROPERTIES:
:ID: unique-id-123
:CUSTOM_ID: custom-123
:CREATED: [2024-01-01]
:END:
Body content.
`
	doc := parseOrgString(t, content)
	h := doc.Nodes[0].(*ir.Heading)

	if h.Props["ID"] != "unique-id-123" {
		t.Errorf("ID = %q, want %q", h.Props["ID"], "unique-id-123")
	}
	if h.Props["CUSTOM_ID"] != "custom-123" {
		t.Errorf("CUSTOM_ID = %q, want %q", h.Props["CUSTOM_ID"], "custom-123")
	}
}

func TestOrgParser_IDRef(t *testing.T) {
	content := `* Heading
:PROPERTIES:
:ID: test-id-abc
:END:
`
	doc := parseOrgString(t, content)
	h := doc.Nodes[0].(*ir.Heading)

	if !strings.Contains(h.Ref, "::ID:test-id-abc") {
		t.Errorf("ref = %q, should contain ::ID:test-id-abc", h.Ref)
	}
}

func TestOrgParser_OutlineRef(t *testing.T) {
	content := `* Heading Without ID
Body.
`
	doc := parseOrgString(t, content)
	h := doc.Nodes[0].(*ir.Heading)

	if !strings.Contains(h.Ref, "::/") {
		t.Errorf("ref = %q, should contain ::/ for outline ref", h.Ref)
	}
}

func TestOrgParser_NestedHeadings(t *testing.T) {
	content := `* Level 1
** Level 2
*** Level 3
** Another Level 2
* Another Level 1
`
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 2 {
		t.Fatalf("expected 2 top-level headings, got %d", len(doc.Nodes))
	}

	h1 := doc.Nodes[0].(*ir.Heading)
	if len(h1.Children) != 2 {
		t.Errorf("first heading should have 2 children, got %d", len(h1.Children))
	}

	if len(h1.Children) > 0 {
		h2 := h1.Children[0].(*ir.Heading)
		if len(h2.Children) != 1 {
			t.Errorf("first level-2 heading should have 1 child, got %d", len(h2.Children))
		}
	}
}

func TestOrgParser_Title(t *testing.T) {
	content := `#+TITLE: My Document Title
* Heading
`
	doc := parseOrgString(t, content)

	if doc.Meta.Title != "My Document Title" {
		t.Errorf("title = %q, want %q", doc.Meta.Title, "My Document Title")
	}
}

func TestOrgParser_MultipleKeywords(t *testing.T) {
	content := `#+TITLE: Document
#+AUTHOR: Test Author
#+DATE: 2024-01-01
* Heading
`
	doc := parseOrgString(t, content)

	if doc.Meta.Title != "Document" {
		t.Errorf("title = %q, want %q", doc.Meta.Title, "Document")
	}
}

func TestOrgParser_SpecialCharsInTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"colon", "* Heading: with colon\n", "Heading: with colon"},
		{"parens", "* Heading (with parens)\n", "Heading (with parens)"},
		{"brackets", "* Heading [with brackets]\n", "Heading [with brackets]"},
		{"quotes", `* Heading "with quotes"` + "\n", `Heading "with quotes"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseOrgString(t, tt.content)
			h := doc.Nodes[0].(*ir.Heading)
			if h.Title != tt.want {
				t.Errorf("title = %q, want %q", h.Title, tt.want)
			}
		})
	}
}

func TestOrgParser_Priority(t *testing.T) {
	content := `* TODO [#A] High priority task
* TODO [#B] Medium priority
* TODO [#C] Low priority
`
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(doc.Nodes))
	}
}

func TestOrgParser_EmptyFile(t *testing.T) {
	content := ``
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 0 {
		t.Errorf("expected 0 headings for empty file, got %d", len(doc.Nodes))
	}
}

func TestOrgParser_OnlyKeywords(t *testing.T) {
	content := `#+TITLE: Only Keywords
#+AUTHOR: Test
`
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 0 {
		t.Errorf("expected 0 headings, got %d", len(doc.Nodes))
	}
	if doc.Meta.Title != "Only Keywords" {
		t.Errorf("title = %q, want %q", doc.Meta.Title, "Only Keywords")
	}
}

func TestOrgParser_CodeBlock(t *testing.T) {
	content := `* Heading with code
#+BEGIN_SRC python
def hello():
    print("Hello")
#+END_SRC
`
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestOrgParser_Table(t *testing.T) {
	content := `* Heading with table
| Name | Age |
|------+-----|
| John | 30  |
| Jane | 25  |
`
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestOrgParser_Links(t *testing.T) {
	content := `* Heading with [[https://example.com][link]]
Body with [[file:other.org][file link]].
`
	doc := parseOrgString(t, content)

	h := doc.Nodes[0].(*ir.Heading)
	if !strings.Contains(h.Title, "link") {
		t.Errorf("title should contain link text, got %q", h.Title)
	}
}

func TestOrgParser_Lists(t *testing.T) {
	content := `* Heading with list
- Item 1
- Item 2
  - Nested item
- Item 3
`
	doc := parseOrgString(t, content)

	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(doc.Nodes))
	}
}

func TestOrgParser_SHA256(t *testing.T) {
	content := `* Heading
Body.
`
	doc := parseOrgString(t, content)

	if doc.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}
	if len(doc.SHA256) != 64 {
		t.Errorf("SHA256 should be 64 chars, got %d", len(doc.SHA256))
	}
}

func TestOrgParser_DocType(t *testing.T) {
	content := `* Heading
`
	doc := parseOrgString(t, content)

	if doc.DocType != ir.DocTypeOrg {
		t.Errorf("DocType = %v, want %v", doc.DocType, ir.DocTypeOrg)
	}
}

func TestOrgParser_RealWorgFile(t *testing.T) {
	// Opt-in corpus test: point ORGX_WORG_FILE at a real worg org
	// file (e.g. worg/todo.org) to parse something battle-hardened.
	worgPath := os.Getenv("ORGX_WORG_FILE")
	if worgPath == "" {
		t.Skip("set ORGX_WORG_FILE to run the worg corpus test")
	}

	doc, err := ParseFile(worgPath)
	if err != nil {
		t.Fatalf("failed to parse worg file: %v", err)
	}

	if len(doc.Nodes) == 0 {
		t.Error("expected headings in worg file")
	}

	hasTopLevel := false
	for _, n := range doc.Nodes {
		if h, ok := n.(*ir.Heading); ok {
			if h.Level == 1 {
				hasTopLevel = true
			}
		}
	}
	if !hasTopLevel {
		t.Error("expected top-level headings")
	}
}

func parseOrgString(t *testing.T, content string) *ir.Document {
	t.Helper()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.org")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	return doc
}
