package parser

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

type MarkdownParser struct{}

func (p *MarkdownParser) CanParse(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

func (p *MarkdownParser) Parse(path string, content []byte) (*ir.Document, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			&frontmatter.Extender{},
		),
	)

	ctx := parser.NewContext()
	reader := text.NewReader(content)
	doc := md.Parser().Parse(reader, parser.WithContext(ctx))

	irDoc := &ir.Document{
		Path:    path,
		SHA256:  computeSHA256(content),
		DocType: ir.DocTypeMarkdown,
		Meta:    ir.DocumentMeta{},
		Nodes:   []ir.Node{},
	}

	fm := extractFrontmatter(ctx)
	if fm != nil {
		irDoc.Meta.Frontmatter = fm
		if title, ok := fm["title"].(string); ok {
			irDoc.Meta.Title = title
		}
	}

	headings := extractMdHeadings(path, doc, content)
	for _, h := range headings {
		irDoc.Nodes = append(irDoc.Nodes, h)
	}

	return irDoc, nil
}

func extractFrontmatter(ctx parser.Context) map[string]any {
	d := frontmatter.Get(ctx)
	if d == nil {
		return nil
	}

	var meta map[string]any
	if err := d.Decode(&meta); err != nil {
		return nil
	}
	return meta
}

func extractMdHeadings(path string, doc ast.Node, content []byte) []*ir.Heading {
	var headings []*ir.Heading
	var stack []*ir.Heading
	headingIndex := 0

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if h, ok := n.(*ast.Heading); ok {
			heading := convertMdHeading(path, h, content, headingIndex)
			headingIndex++

			for len(stack) > 0 && stack[len(stack)-1].Level >= heading.Level {
				stack = stack[:len(stack)-1]
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, heading)
			} else {
				headings = append(headings, heading)
			}

			stack = append(stack, heading)
		}

		return ast.WalkContinue, nil
	})

	return headings
}

func convertMdHeading(path string, h *ast.Heading, content []byte, headingIndex int) *ir.Heading {
	title := extractMdText(h, content)

	startOffset := 0
	endOffset := 0
	if h.Lines().Len() > 0 {
		startOffset = h.Lines().At(0).Start
		endOffset = h.Lines().At(h.Lines().Len() - 1).Stop
	}

	ref := buildMdRef(path, title, startOffset)

	bodyRaw := extractMdHeadingBody(h, content)
	links := extractMdLinks(h, content)

	return &ir.Heading{
		Type:  ir.NodeTypeHeading,
		Ref:   ref,
		Level: h.Level,
		Title: title,
		Body: ir.Body{
			Raw: bodyRaw,
		},
		Links: links,
		Span: ir.Span{
			Start: startOffset, // byte offset, not line number
			End:   endOffset,
		},
	}
}

func extractMdLinks(h *ast.Heading, content []byte) []*ir.Link {
	var links []*ir.Link

	for sibling := h.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		if _, ok := sibling.(*ast.Heading); ok {
			break
		}
		extractMdLinksFromNode(sibling, content, &links)
	}

	return links
}

func extractMdLinksFromNode(n ast.Node, content []byte, links *[]*ir.Link) {
	ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := child.(*ast.Link); ok {
			dest := string(link.Destination)
			l := convertMdLink(dest, extractMdText(link, content))
			if l != nil {
				*links = append(*links, l)
			}
		}

		if autoLink, ok := child.(*ast.AutoLink); ok {
			dest := string(autoLink.URL(content))
			l := convertMdLink(dest, dest)
			if l != nil {
				*links = append(*links, l)
			}
		}

		return ast.WalkContinue, nil
	})
}

func convertMdLink(dest, desc string) *ir.Link {
	var kind ir.LinkKind

	switch {
	case strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://"):
		kind = ir.LinkKindHTTP
	case strings.HasSuffix(dest, ".md") || strings.HasSuffix(dest, ".org"):
		kind = ir.LinkKindFile
	case strings.HasPrefix(dest, "./") || strings.HasPrefix(dest, "../"):
		kind = ir.LinkKindFile
	default:
		return nil
	}

	return &ir.Link{
		Type:   ir.NodeTypeLink,
		Kind:   kind,
		Target: dest,
		Desc:   desc,
	}
}

func buildMdRef(path, title string, offset int) string {
	// Include offset in hash to ensure unique refs even for duplicate titles
	data := fmt.Sprintf("%s:%d:%s", path, offset, title)
	hash := sha256.Sum256([]byte(data))
	shortHash := fmt.Sprintf("%x", hash[:4])
	return fmt.Sprintf("%s::H:%s", path, shortHash)
}

func extractMdText(n ast.Node, content []byte) string {
	var buf bytes.Buffer
	ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Segment.Value(content))
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

func extractMdHeadingBody(h *ast.Heading, content []byte) string {
	var parts []string
	for sibling := h.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		if _, ok := sibling.(*ast.Heading); ok {
			break
		}
		parts = append(parts, renderMdNode(sibling, content))
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func renderMdNode(n ast.Node, content []byte) string {
	switch v := n.(type) {
	case *ast.Paragraph:
		return extractMdText(v, content)
	case *ast.FencedCodeBlock:
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(content))
		}
		lang := string(v.Language(content))
		return fmt.Sprintf("```%s\n%s```", lang, buf.String())
	case *ast.CodeBlock:
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(content))
		}
		return fmt.Sprintf("```\n%s```", buf.String())
	case *ast.List:
		var items []string
		for child := v.FirstChild(); child != nil; child = child.NextSibling() {
			if item, ok := child.(*ast.ListItem); ok {
				items = append(items, "- "+extractMdText(item, content))
			}
		}
		return strings.Join(items, "\n")
	case *ast.Blockquote:
		return "> " + extractMdText(v, content)
	default:
		return ""
	}
}
