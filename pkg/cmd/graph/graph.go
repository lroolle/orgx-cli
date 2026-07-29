// Package graph derives the vault's link graph: file-level nodes,
// id-link edges, and the broken references the files imply. The
// graph is computed, never stored — the org files are the database.
package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

type GraphOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Root string
}

type Edge struct {
	From string `json:"from"` // source node id
	To   string `json:"to"`   // target node id
}

type Broken struct {
	From   string `json:"from"`   // source node id
	Target string `json:"target"` // id that resolves to no node
}

type Result struct {
	Root   string          `json:"root"`
	Nodes  []roam.NodeMeta `json:"nodes"`
	Edges  []Edge          `json:"edges"`
	Broken []Broken        `json:"broken,omitempty"`
}

func NewCmdGraph(f *cmdutil.Factory, runF func(*GraphOptions) error) *cobra.Command {
	opts := &GraphOptions{IO: f.IOStreams}

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Derive the vault's node/edge graph",
		Long: `Walk the vault and emit its graph: every node (org file with a
file-level :ID:), every id-link edge between nodes, and the broken
references (links to ids no node carries).

The graph is derived on demand, never cached in the vault — pipe it
to jq, a visualizer, or an agent.

Examples:
  orgx graph --json
  orgx graph --json | jq '.edges | length'
  orgx graph --json | jq '.broken'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, _ := cmd.Flags().GetString("workspace")
			rootFlag, _ := cmd.Flags().GetString("root")
			root, err := roam.ResolveRoot(config.LoadOrDefault(), ws, rootFlag)
			if err != nil {
				return err
			}
			opts.Root = root
			if runF != nil {
				return runF(opts)
			}
			return graphRun(opts)
		},
	}
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"nodes", "edges", "broken"})
	return cmd
}

func graphRun(opts *GraphOptions) error {
	nodes, _, err := roam.Scan(opts.Root)
	if err != nil {
		return fmt.Errorf("scan %s: %w", opts.Root, err)
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

	result := Result{Root: opts.Root, Nodes: nodes, Edges: edges, Broken: broken}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}
	fmt.Fprintf(opts.IO.Out, "%d nodes, %d edges", len(result.Nodes), len(result.Edges))
	if len(broken) > 0 {
		fmt.Fprintf(opts.IO.Out, ", %d broken links", len(broken))
	}
	fmt.Fprintln(opts.IO.Out)
	for _, b := range broken {
		fmt.Fprintf(opts.IO.Out, "  ! %s -> %s (no such node)\n", b.From, b.Target)
	}
	if len(edges) == 0 && len(nodes) > 0 {
		fmt.Fprintln(opts.IO.Out, "# → link nodes with [[id:<node-id>]] in entries and pages")
	}
	return nil
}

// idLinks parses one file and returns the ids it links to.
func idLinks(path string) []string {
	doc, err := parser.ParseFile(path)
	if err != nil {
		return nil
	}
	var ids []string
	var walk func(nodes []ir.Node)
	walk = func(nodes []ir.Node) {
		for _, n := range nodes {
			h, ok := n.(*ir.Heading)
			if !ok {
				continue
			}
			for _, l := range h.Links {
				if l.Kind == ir.LinkKindID {
					ids = append(ids, strings.TrimPrefix(l.Target, "id:"))
				}
			}
			walk(h.Children)
		}
	}
	walk(doc.Nodes)
	return ids
}
