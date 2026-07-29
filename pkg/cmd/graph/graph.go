// Package graph emits the vault's link graph: file-level nodes,
// id-link edges, and the broken references the files imply. The
// derivation lives in pkg/roam (BuildGraph) so the CLI and the web
// preview report the same graph.
package graph

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

type GraphOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Root string
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
	result, err := roam.BuildGraph(opts.Root)
	if err != nil {
		return fmt.Errorf("scan %s: %w", opts.Root, err)
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}
	fmt.Fprintf(opts.IO.Out, "%d nodes, %d edges", len(result.Nodes), len(result.Edges))
	if len(result.Broken) > 0 {
		fmt.Fprintf(opts.IO.Out, ", %d broken links", len(result.Broken))
	}
	fmt.Fprintln(opts.IO.Out)
	for _, b := range result.Broken {
		fmt.Fprintf(opts.IO.Out, "  ! %s -> %s (no such node)\n", b.From, b.Target)
	}
	if len(result.Edges) == 0 && len(result.Nodes) > 0 {
		fmt.Fprintln(opts.IO.Out, "# → link nodes with [[id:<node-id>]] in entries and pages")
	}
	return nil
}
