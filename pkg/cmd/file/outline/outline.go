package outline

import (
	"fmt"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type OutlineOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path     string
	MaxDepth int
}

func NewCmdOutline(f *cmdutil.Factory, runF func(*OutlineOptions) error) *cobra.Command {
	opts := &OutlineOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "outline <path>",
		Short: "Show file outline",
		Long:  "Display the heading structure of an org or markdown file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Path = args[0]

			if runF != nil {
				return runF(opts)
			}
			return outlineRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.MaxDepth, "max-depth", 0, "Maximum heading depth (0 = unlimited)")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, defaultFields)

	return cmd
}

var defaultFields = []string{
	"ref",
	"level",
	"title",
	"todo",
	"tags",
}

type OutlineEntry struct {
	Ref      string         `json:"ref"`
	Level    int            `json:"level"`
	Title    string         `json:"title"`
	Todo     string         `json:"todo,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Children []OutlineEntry `json:"children,omitempty"`
}

func outlineRun(opts *OutlineOptions) error {
	doc, err := parser.ParseFile(opts.Path)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	entries := buildOutline(doc.Nodes, opts.MaxDepth, 1)

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, entries)
	}

	return printOutline(opts.IO, entries, 0)
}

func buildOutline(nodes []ir.Node, maxDepth, currentDepth int) []OutlineEntry {
	var entries []OutlineEntry

	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			if maxDepth > 0 && currentDepth > maxDepth {
				continue
			}

			entry := OutlineEntry{
				Ref:   h.Ref,
				Level: h.Level,
				Title: h.Title,
				Todo:  h.Todo,
				Tags:  h.Tags,
			}

			if len(h.Children) > 0 {
				childNodes := make([]ir.Node, len(h.Children))
				copy(childNodes, h.Children)
				entry.Children = buildOutline(childNodes, maxDepth, currentDepth+1)
			}

			entries = append(entries, entry)
		}
	}

	return entries
}

func printOutline(io *iostreams.IOStreams, entries []OutlineEntry, indent int) error {
	for _, e := range entries {
		prefix := strings.Repeat("  ", indent)
		stars := strings.Repeat("*", e.Level)

		line := fmt.Sprintf("%s%s", prefix, stars)
		if e.Todo != "" {
			line += " " + e.Todo
		}
		line += " " + e.Title
		if len(e.Tags) > 0 {
			line += " :" + strings.Join(e.Tags, ":") + ":"
		}

		fmt.Fprintln(io.Out, line)

		if len(e.Children) > 0 {
			printOutline(io, e.Children, indent+1)
		}
	}
	return nil
}
