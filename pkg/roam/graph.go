package roam

import (
	"sort"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
)

// Edge is one id-link between two nodes.
type Edge struct {
	From string `json:"from"` // source node id
	To   string `json:"to"`   // target node id
}

// Broken is an id link that resolves to no node under the root.
type Broken struct {
	From   string `json:"from"`   // source node id
	Target string `json:"target"` // id that resolves to no node
}

// Graph is the vault's derived link graph. It is computed from the
// files on every call and never stored — the org files are the
// database (see ARCHITECTURE.md).
type Graph struct {
	Root   string     `json:"root"`
	Nodes  []NodeMeta `json:"nodes"`
	Edges  []Edge     `json:"edges"`
	Broken []Broken   `json:"broken,omitempty"`
}

// BuildGraph scans root and derives nodes, deduplicated id-link
// edges, and broken references.
func BuildGraph(root string) (Graph, error) {
	nodes, _, err := Scan(root)
	if err != nil {
		return Graph{Root: root}, err
	}
	known := map[string]bool{}
	for _, n := range nodes {
		known[n.ID] = true
	}

	var edges []Edge
	var broken []Broken
	seen := map[Edge]bool{}
	for _, n := range nodes {
		for _, target := range idLinks(n.Path) {
			if known[target] {
				e := Edge{From: n.ID, To: target}
				if !seen[e] {
					seen[e] = true
					edges = append(edges, e)
				}
				continue
			}
			broken = append(broken, Broken{From: n.ID, Target: target})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].From+edges[i].To < edges[j].From+edges[j].To
	})
	return Graph{Root: root, Nodes: nodes, Edges: edges, Broken: broken}, nil
}

// Node returns the node carrying id, or nil.
func (g *Graph) Node(id string) *NodeMeta {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}

// Backlinks returns the nodes that link to id, title-sorted.
func (g *Graph) Backlinks(id string) []NodeMeta {
	var sources []NodeMeta
	seen := map[string]bool{}
	for _, e := range g.Edges {
		if e.To != id || seen[e.From] {
			continue
		}
		seen[e.From] = true
		if n := g.Node(e.From); n != nil {
			sources = append(sources, *n)
		}
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Title < sources[j].Title })
	return sources
}

// idLinks parses one file and returns the ids it links to.
func idLinks(path string) []string {
	doc, err := parser.ParseFile(path)
	if err != nil {
		return nil
	}
	var ids []string
	add := func(l *ir.Link) {
		if l.Kind == ir.LinkKindID {
			ids = append(ids, strings.TrimPrefix(l.Target, "id:"))
		}
	}
	var walk func(nodes []ir.Node)
	walk = func(nodes []ir.Node) {
		for _, n := range nodes {
			switch v := n.(type) {
			case *ir.Link: // file-level preamble link
				add(v)
			case *ir.Heading:
				for _, l := range v.Links {
					add(l)
				}
				walk(v.Children)
			}
		}
	}
	walk(doc.Nodes)
	return ids
}
