package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/niklasfasching/go-org/org"
)

type OrgParser struct {
	StateKeywords []string // if set, uses these; otherwise loads from config
}

func (p *OrgParser) CanParse(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".org")
}

func (p *OrgParser) Parse(path string, content []byte) (*ir.Document, error) {
	orgConf := org.New()
	doc := orgConf.Parse(strings.NewReader(string(content)), path)

	stateKeywords := p.StateKeywords
	if len(stateKeywords) == 0 {
		cfg := config.LoadOrDefault()
		stateKeywords = cfg.GetStateKeywords()
	}

	irDoc := &ir.Document{
		Path:    path,
		SHA256:  computeSHA256(content),
		DocType: ir.DocTypeOrg,
		Meta:    ir.DocumentMeta{},
		Nodes:   []ir.Node{},
	}

	if title := extractOrgTitle(doc); title != "" {
		irDoc.Meta.Title = title
	}

	lines := strings.Split(string(content), "\n")
	headings := extractOrgHeadings(path, doc.Nodes, lines, stateKeywords)
	for _, h := range headings {
		irDoc.Nodes = append(irDoc.Nodes, h)
	}

	return irDoc, nil
}

func extractOrgTitle(doc *org.Document) string {
	if title, ok := doc.BufferSettings["TITLE"]; ok {
		return title
	}
	return ""
}

func extractOrgHeadings(path string, nodes []org.Node, lines []string, stateKeywords []string) []*ir.Heading {
	var headings []*ir.Heading
	lineTracker := &lineTracker{lines: lines, lastLine: 0}

	for _, node := range nodes {
		if h, ok := node.(org.Headline); ok {
			heading := convertOrgHeadline(path, h, lines, lineTracker, stateKeywords)
			headings = append(headings, heading)

			children := extractOrgHeadingsWithTracker(path, h.Children, lines, lineTracker, stateKeywords)
			for _, child := range children {
				heading.Children = append(heading.Children, child)
			}
		}
	}

	return headings
}

type lineTracker struct {
	lines    []string
	lastLine int
}

func (t *lineTracker) findHeadingLine(level int, title string) int {
	prefix := strings.Repeat("*", level) + " "
	for i := t.lastLine; i < len(t.lines); i++ {
		line := t.lines[i]
		if strings.HasPrefix(line, prefix) && strings.Contains(line, title) {
			t.lastLine = i + 1
			return i + 1 // 1-based
		}
	}
	return 0
}

func extractOrgHeadingsWithTracker(path string, nodes []org.Node, lines []string, tracker *lineTracker, stateKeywords []string) []*ir.Heading {
	var headings []*ir.Heading

	for _, node := range nodes {
		if h, ok := node.(org.Headline); ok {
			heading := convertOrgHeadline(path, h, lines, tracker, stateKeywords)
			headings = append(headings, heading)

			children := extractOrgHeadingsWithTracker(path, h.Children, lines, tracker, stateKeywords)
			for _, child := range children {
				heading.Children = append(heading.Children, child)
			}
		}
	}

	return headings
}

func convertOrgHeadline(path string, h org.Headline, lines []string, tracker *lineTracker, stateKeywords []string) *ir.Heading {
	title := renderOrgNodes(h.Title)
	todo := h.Status

	// go-org only recognizes built-in states (TODO, DONE).
	// Custom states like IDEA, STRT become part of the title.
	// Extract them here.
	if todo == "" {
		todo, title = extractCustomState(title, stateKeywords)
	}

	props := extractProperties(h)
	ref := buildOrgRef(path, props, title, stateKeywords)

	scheduled, deadline, closed := extractScheduling(h.Children)
	logbook := extractLogbook(h.Children)
	bodyRaw := extractHeadlineBody(h)
	// Links live in the title too — journal entries are headings, and
	// "* 14:30 worked on [[id:x]]" must backlink like body text does.
	links := append(extractOrgLinks(h.Title), extractOrgLinks(h.Children)...)

	lineNum := tracker.findHeadingLine(h.Lvl, title)

	return &ir.Heading{
		Type:      ir.NodeTypeHeading,
		Ref:       ref,
		Level:     h.Lvl,
		Title:     title,
		Todo:      todo,
		Tags:      h.Tags,
		Props:     props,
		Scheduled: scheduled,
		Deadline:  deadline,
		Closed:    closed,
		Logbook:   logbook,
		Body: ir.Body{
			Raw: bodyRaw,
		},
		Links: links,
		Span: ir.Span{
			Start: lineNum,
			End:   lineNum,
		},
	}
}

func extractCustomState(title string, stateKeywords []string) (state, newTitle string) {
	for _, kw := range stateKeywords {
		prefix := kw + " "
		if strings.HasPrefix(title, prefix) {
			return kw, strings.TrimPrefix(title, prefix)
		}
	}
	return "", title
}

func extractProperties(h org.Headline) map[string]string {
	props := make(map[string]string)

	// go-org puts properties in h.Properties when property drawer is
	// immediately after headline. But when planning line (CLOSED/SCHEDULED/
	// DEADLINE) is present, h.Properties is nil and the property drawer
	// becomes an org.PropertyDrawer child node.
	if h.Properties != nil {
		for _, kv := range h.Properties.Properties {
			props[kv[0]] = kv[1]
		}
		return props
	}

	// Check children for PropertyDrawer
	for _, child := range h.Children {
		if pd, ok := child.(org.PropertyDrawer); ok {
			for _, kv := range pd.Properties {
				props[kv[0]] = kv[1]
			}
			break
		}
	}
	return props
}

func extractOrgLinks(nodes []org.Node) []*ir.Link {
	var links []*ir.Link
	for _, node := range nodes {
		extractLinksFromNode(node, &links)
	}
	return links
}

func extractLinksFromNode(node org.Node, links *[]*ir.Link) {
	switch v := node.(type) {
	case org.RegularLink:
		link := convertOrgLink(v)
		if link != nil {
			*links = append(*links, link)
		}
	case org.Paragraph:
		for _, child := range v.Children {
			extractLinksFromNode(child, links)
		}
	case org.List:
		for _, item := range v.Items {
			extractLinksFromNode(item, links)
		}
	case org.ListItem:
		for _, child := range v.Children {
			extractLinksFromNode(child, links)
		}
	case org.Table:
		for _, row := range v.Rows {
			for _, col := range row.Columns {
				for _, child := range col.Children {
					extractLinksFromNode(child, links)
				}
			}
		}
	}
}

func convertOrgLink(l org.RegularLink) *ir.Link {
	var kind ir.LinkKind
	switch l.Protocol {
	case "file":
		kind = ir.LinkKindFile
	case "id":
		kind = ir.LinkKindID
	case "http", "https":
		kind = ir.LinkKindHTTP
	case "":
		if strings.HasPrefix(l.URL, "./") || strings.HasPrefix(l.URL, "../") || strings.HasSuffix(l.URL, ".org") || strings.HasSuffix(l.URL, ".md") {
			kind = ir.LinkKindFile
		} else {
			return nil
		}
	default:
		return nil
	}

	desc := ""
	if l.Description != nil {
		desc = renderOrgNodes(l.Description)
	}

	return &ir.Link{
		Type:   ir.NodeTypeLink,
		Kind:   kind,
		Target: l.URL,
		Desc:   desc,
	}
}

var planningLineRe = regexp.MustCompile(`^(CLOSED|SCHEDULED|DEADLINE):`)
var planningTokenRe = regexp.MustCompile(`(CLOSED|SCHEDULED|DEADLINE):\s*(\[[^\]]+\]|<[^>]+>)`)

func extractScheduling(nodes []org.Node) (scheduled, deadline, closed string) {
	// Planning line must be first child(ren) before any drawer or body content.
	// Once we see a drawer or non-planning paragraph, stop looking.
	for _, node := range nodes {
		// go-org parses planning as org.Keyword when immediately after headline
		if kw, ok := node.(org.Keyword); ok {
			switch kw.Key {
			case "SCHEDULED":
				scheduled = kw.Value
			case "DEADLINE":
				deadline = kw.Value
			case "CLOSED":
				closed = kw.Value
			}
			continue
		}

		// Stop at any drawer (PropertyDrawer, LOGBOOK, etc.)
		if _, ok := node.(org.PropertyDrawer); ok {
			break
		}
		if _, ok := node.(org.Drawer); ok {
			break
		}

		// go-org may parse planning line as org.Paragraph
		if para, ok := node.(org.Paragraph); ok {
			text := strings.TrimSpace(renderOrgNodes(para.Children))
			// Guard: line must START with a planning keyword to be a planning line
			if !planningLineRe.MatchString(text) {
				// Not a planning line, this is body content - stop
				break
			}
			// Extract ALL planning tokens from the line (unanchored)
			matches := planningTokenRe.FindAllStringSubmatch(text, -1)
			for _, m := range matches {
				switch m[1] {
				case "SCHEDULED":
					if scheduled == "" {
						scheduled = m[2]
					}
				case "DEADLINE":
					if deadline == "" {
						deadline = m[2]
					}
				case "CLOSED":
					if closed == "" {
						closed = m[2]
					}
				}
			}
			continue
		}

		// Any other node type means we're past planning
		break
	}
	return
}

var stateChangeRe = regexp.MustCompile(`State\s+"([^"]*)"\s+from\s+"([^"]*)"\s+(\[.+?\])`)

func extractLogbook(nodes []org.Node) []ir.StateChange {
	var changes []ir.StateChange
	for _, node := range nodes {
		if drawer, ok := node.(org.Drawer); ok && drawer.Name == "LOGBOOK" {
			for _, child := range drawer.Children {
				text := extractTextFromNode(child)
				matches := stateChangeRe.FindAllStringSubmatch(text, -1)
				for _, m := range matches {
					changes = append(changes, ir.StateChange{
						NewState:  m[1],
						OldState:  m[2],
						Timestamp: m[3],
					})
				}
			}
		}
	}
	return changes
}

func extractTextFromNode(node org.Node) string {
	switch v := node.(type) {
	case org.Paragraph:
		return renderOrgNodes(v.Children)
	case org.List:
		var parts []string
		for _, item := range v.Items {
			parts = append(parts, extractTextFromNode(item))
		}
		return strings.Join(parts, "\n")
	case org.ListItem:
		return renderOrgNodes(v.Children)
	case org.Text:
		return v.Content
	default:
		return ""
	}
}

func buildOrgRef(path string, props map[string]string, title string, stateKeywords []string) string {
	if id, ok := props["ID"]; ok {
		return fmt.Sprintf("%s::ID:%s", path, id)
	}
	return fmt.Sprintf("%s::/%s", path, sanitizeOutlinePath(title))
}

func sanitizeOutlinePath(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.TrimSpace(s)
	return s
}

func renderOrgNodes(nodes []org.Node) string {
	var parts []string
	for _, n := range nodes {
		switch v := n.(type) {
		case org.Text:
			parts = append(parts, v.Content)
		case org.Emphasis:
			parts = append(parts, renderOrgNodes(v.Content))
		case org.RegularLink:
			if v.Description != nil {
				parts = append(parts, renderOrgNodes(v.Description))
			} else {
				parts = append(parts, v.URL)
			}
		default:
			if s := fmt.Sprintf("%v", n); s != "" {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, "")
}

func extractHeadlineBody(h org.Headline) string {
	var bodyParts []string
	for _, child := range h.Children {
		if _, isHeadline := child.(org.Headline); !isHeadline {
			bodyParts = append(bodyParts, renderOrgNode(child))
		}
	}
	result := strings.Join(bodyParts, "\n")
	return strings.TrimSpace(result)
}

func renderOrgNode(n org.Node) string {
	switch v := n.(type) {
	case org.Paragraph:
		return renderOrgNodes(v.Children)
	case org.Block:
		var lines []string
		for _, child := range v.Children {
			lines = append(lines, renderOrgNode(child))
		}
		return fmt.Sprintf("#+BEGIN_%s\n%s\n#+END_%s", v.Name, strings.Join(lines, "\n"), v.Name)
	case org.List:
		var items []string
		for _, item := range v.Items {
			items = append(items, renderOrgNode(item))
		}
		return strings.Join(items, "\n")
	case org.ListItem:
		return fmt.Sprintf("%s %s", v.Bullet, renderOrgNodes(v.Children))
	case org.Table:
		return renderOrgTable(v)
	case org.Keyword:
		return ""
	case org.PropertyDrawer:
		return ""
	default:
		return ""
	}
}

func renderOrgTable(t org.Table) string {
	var rows []string
	for _, row := range t.Rows {
		if row.IsSpecial {
			continue
		}
		var cells []string
		for _, col := range row.Columns {
			cells = append(cells, renderOrgNodes(col.Children))
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
	}
	return strings.Join(rows, "\n")
}
