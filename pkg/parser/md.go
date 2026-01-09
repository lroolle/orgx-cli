package parser

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/textutil"
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
	lines := newLineIndex(content)

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

	headings := extractMdHeadings(path, doc, content, lines)
	for _, h := range headings {
		irDoc.Nodes = append(irDoc.Nodes, h)
	}

	return irDoc, nil
}

type lineIndex struct {
	starts []int
}

func newLineIndex(content []byte) *lineIndex {
	starts := []int{0}
	for i, b := range content {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &lineIndex{starts: starts}
}

func (li *lineIndex) lineAtByteOffset(offset int) int {
	if offset <= 0 {
		return 1
	}
	idx := sort.Search(len(li.starts), func(i int) bool {
		return li.starts[i] > offset
	}) - 1
	if idx < 0 {
		return 1
	}
	return idx + 1
}

func (li *lineIndex) lineContent(lineNum int, content []byte) []byte {
	idx := lineNum - 1 // 1-based to 0-based
	if idx < 0 || idx >= len(li.starts) {
		return nil
	}
	start := li.starts[idx]
	end := len(content)
	if idx+1 < len(li.starts) {
		end = li.starts[idx+1]
	}
	// Trim trailing newline
	if end > start && end <= len(content) && content[end-1] == '\n' {
		end--
	}
	if end > start && end <= len(content) && content[end-1] == '\r' {
		end--
	}
	if end < start {
		return nil
	}
	return content[start:end]
}

func (li *lineIndex) lineEndOffset(lineNum int, content []byte) int {
	idx := lineNum - 1 // 1-based to 0-based
	if idx < 0 || idx >= len(li.starts) {
		return len(content)
	}
	if idx+1 < len(li.starts) {
		return li.starts[idx+1]
	}
	return len(content)
}

func isATXHeading(line []byte) bool {
	if len(line) == 0 {
		return false
	}
	// ATX headings allow up to 3 leading spaces before #
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	return i < len(line) && line[i] == '#'
}

func isSetextUnderline(line []byte) bool {
	if len(line) == 0 {
		return false
	}
	// Goldmark allows up to 3 leading spaces
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) {
		return false
	}
	// First non-space char must be = or -
	char := line[i]
	if char != '=' && char != '-' {
		return false
	}
	// Must have contiguous run of same char (no interspersed spaces)
	for i < len(line) && line[i] == char {
		i++
	}
	// Rest must be only trailing whitespace
	for i < len(line) {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
		i++
	}
	return true
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

func extractMdHeadings(path string, doc ast.Node, content []byte, lines *lineIndex) []*ir.Heading {
	var headings []*ir.Heading
	var stack []*ir.Heading
	headingIndex := 0

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if h, ok := n.(*ast.Heading); ok {
			heading := convertMdHeading(path, h, content, lines, headingIndex)
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

func convertMdHeading(path string, h *ast.Heading, content []byte, lines *lineIndex, headingIndex int) *ir.Heading {
	title := extractMdText(h, content)

	startOffset := 0
	endOffset := 0
	if h.Lines().Len() > 0 {
		startOffset = h.Lines().At(0).Start
		endOffset = h.Lines().At(h.Lines().Len() - 1).Stop
	}

	startLine := lines.lineAtByteOffset(startOffset)
	endLineOffset := endOffset
	if endLineOffset > 0 {
		endLineOffset--
	}
	endLine := lines.lineAtByteOffset(endLineOffset)

	// For setext headings, goldmark's h.Lines() only covers the title line,
	// not the underline. Check if next line is a setext underline and adjust.
	// Only check if this is NOT an ATX heading (ATX headings start with # after up to 3 spaces).
	titleLineContent := lines.lineContent(startLine, content)
	isATX := isATXHeading(titleLineContent)
	if !isATX {
		nextLineContent := lines.lineContent(endLine+1, content)
		if isSetextUnderline(nextLineContent) {
			endLine++
			// Update endOffset to include the underline for ID extraction
			endOffset = lines.lineEndOffset(endLine, content)
		}
	}

	id := extractOrgxIDFromMarkdownHeading(content, startOffset, endOffset)

	ref := ""
	props := map[string]string(nil)
	if id != "" {
		ref = fmt.Sprintf("%s::ID:%s", path, id)
		props = map[string]string{"ID": id}
	} else {
		ref = buildMdRef(path, title, startOffset)
	}

	bodyRaw := extractMdHeadingBody(h, content)
	links := extractMdLinks(h, content)

	return &ir.Heading{
		Type:  ir.NodeTypeHeading,
		Ref:   ref,
		Level: h.Level,
		Title: title,
		Props: props,
		Body: ir.Body{
			Raw: bodyRaw,
		},
		Links: links,
		Span: ir.Span{
			Start: startLine,
			End:   endLine,
		},
	}
}

func extractOrgxIDFromMarkdownHeading(content []byte, headingStartOffset, headingEndOffset int) string {
	if headingStartOffset < 0 || headingStartOffset >= len(content) {
		return ""
	}
	if headingEndOffset < headingStartOffset {
		headingEndOffset = headingStartOffset
	}
	if headingEndOffset > len(content) {
		headingEndOffset = len(content)
	}

	if id := extractOrgxIDFromMarkdownLine(content[headingStartOffset:headingEndOffset]); id != "" {
		return id
	}

	pos := headingEndOffset
	maxLinesToScan := 6
	for scanned := 0; scanned < maxLinesToScan && pos < len(content); scanned++ {
		line, nextPos := readLineBytes(content, pos)
		pos = nextPos

		trimmed := strings.TrimSpace(strings.TrimRight(string(line), "\r"))
		if trimmed == "" {
			continue
		}

		if id := extractOrgxIDFromMarkdownLine([]byte(trimmed)); id != "" {
			return id
		}

		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			continue
		}

		break
	}

	return ""
}

func extractOrgxIDFromMarkdownLine(line []byte) string {
	m := textutil.OrgxIDExtractRe.FindSubmatch(line)
	if m == nil {
		return ""
	}
	return string(m[1])
}

func readLineBytes(content []byte, start int) ([]byte, int) {
	if start < 0 {
		start = 0
	}
	if start >= len(content) {
		return nil, len(content)
	}
	end := start
	for end < len(content) && content[end] != '\n' {
		end++
	}
	next := end
	if next < len(content) && content[next] == '\n' {
		next++
	}
	return content[start:end], next
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
