package parser

import (
	"fmt"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/niklasfasching/go-org/org"
)

type OrgParser struct{}

func (p *OrgParser) CanParse(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".org")
}

func (p *OrgParser) Parse(path string, content []byte) (*ir.Document, error) {
	config := org.New()
	doc := config.Parse(strings.NewReader(string(content)), path)

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
	headings := extractOrgHeadings(path, doc.Nodes, lines)
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

func extractOrgHeadings(path string, nodes []org.Node, lines []string) []*ir.Heading {
	var headings []*ir.Heading

	for _, node := range nodes {
		if h, ok := node.(org.Headline); ok {
			heading := convertOrgHeadline(path, h, lines)
			headings = append(headings, heading)

			children := extractOrgHeadings(path, h.Children, lines)
			for _, child := range children {
				heading.Children = append(heading.Children, child)
			}
		}
	}

	return headings
}

func convertOrgHeadline(path string, h org.Headline, lines []string) *ir.Heading {
	ref := buildOrgRef(path, h)

	props := make(map[string]string)
	if h.Properties != nil {
		for _, kv := range h.Properties.Properties {
			props[kv[0]] = kv[1]
		}
	}

	scheduled, deadline := extractScheduling(h.Children)
	bodyRaw := extractHeadlineBody(h)

	return &ir.Heading{
		Type:      ir.NodeTypeHeading,
		Ref:       ref,
		Level:     h.Lvl,
		Title:     renderOrgNodes(h.Title),
		Todo:      h.Status,
		Tags:      h.Tags,
		Props:     props,
		Scheduled: scheduled,
		Deadline:  deadline,
		Body: ir.Body{
			Raw: bodyRaw,
		},
		Span: ir.Span{
			Start: h.Index,
			End:   h.Index,
		},
	}
}

func extractScheduling(nodes []org.Node) (scheduled, deadline string) {
	for _, node := range nodes {
		if kw, ok := node.(org.Keyword); ok {
			switch kw.Key {
			case "SCHEDULED":
				scheduled = kw.Value
			case "DEADLINE":
				deadline = kw.Value
			}
		}
	}
	return
}

func buildOrgRef(path string, h org.Headline) string {
	if h.Properties != nil {
		for _, kv := range h.Properties.Properties {
			if kv[0] == "ID" {
				return fmt.Sprintf("%s::ID:%s", path, kv[1])
			}
		}
	}
	title := renderOrgNodes(h.Title)
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
