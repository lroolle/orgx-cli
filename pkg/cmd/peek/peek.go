package peek

import (
	"fmt"
	"os"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type PeekOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path     string
	MaxDepth int
}

func NewCmdPeek(f *cmdutil.Factory, runF func(*PeekOptions) error) *cobra.Command {
	opts := &PeekOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "peek <path>",
		Short: "Show file structure (low tokens)",
		Long: `Show the heading structure of a file without loading content.
Perfect for understanding what's in a file before diving in.

Examples:
  orgx peek notes.org
  orgx peek ~/org/projects.org --max-depth 2 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Path = args[0]

			if runF != nil {
				return runF(opts)
			}
			return peekRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.MaxDepth, "max-depth", 0, "Maximum heading depth (0 = unlimited)")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, defaultFields)

	return cmd
}

var defaultFields = []string{"path", "lines", "headings"}

type PeekOutput struct {
	Path       string      `json:"path"`
	Lines      int         `json:"lines"`
	Headings   int         `json:"heading_count"`
	TotalCount int         `json:"total_count,omitempty"`
	Entries    []PeekEntry `json:"headings"`
	Truncated  bool        `json:"truncated,omitempty"`
	MaxDepth   int         `json:"-"`
	LastRef    string      `json:"last_ref,omitempty"`
}

type PeekEntry struct {
	Ref   string   `json:"ref"`
	Level int      `json:"level"`
	Title string   `json:"title"`
	Todo  string   `json:"todo,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

func peekRun(opts *PeekOptions) error {
	content, err := os.ReadFile(opts.Path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	doc, err := parser.ParseFile(opts.Path)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	lineCount := strings.Count(string(content), "\n") + 1
	allEntries := collectEntries(doc.Nodes, 0, 1)
	entries := collectEntries(doc.Nodes, opts.MaxDepth, 1)

	truncated := opts.MaxDepth > 0 && len(entries) < len(allEntries)

	var lastRef string
	if len(entries) > 0 {
		lastRef = entries[len(entries)-1].Ref
	}

	output := PeekOutput{
		Path:       opts.Path,
		Lines:      lineCount,
		Headings:   len(entries),
		TotalCount: len(allEntries),
		Entries:    entries,
		Truncated:  truncated,
		MaxDepth:   opts.MaxDepth,
		LastRef:    lastRef,
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, output)
	}

	return printPeek(opts.IO, output)
}

func collectEntries(nodes []ir.Node, maxDepth, currentDepth int) []PeekEntry {
	var entries []PeekEntry

	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			if maxDepth > 0 && currentDepth > maxDepth {
				continue
			}

			entries = append(entries, PeekEntry{
				Ref:   h.Ref,
				Level: h.Level,
				Title: h.Title,
				Todo:  h.Todo,
				Tags:  h.Tags,
			})

			if len(h.Children) > 0 {
				entries = append(entries, collectEntries(h.Children, maxDepth, currentDepth+1)...)
			}
		}
	}

	return entries
}

func printPeek(io *iostreams.IOStreams, output PeekOutput) error {
	fmt.Fprintf(io.Out, "%s (%d lines, %d headings)\n", output.Path, output.Lines, output.Headings)

	for _, e := range output.Entries {
		indent := strings.Repeat("  ", e.Level-1)
		stars := strings.Repeat("*", e.Level)

		line := fmt.Sprintf("%s%s", indent, stars)
		if e.Todo != "" {
			line += " " + e.Todo
		}
		line += " " + e.Title

		refPart := extractRefShort(e.Ref)
		line += "  " + refPart

		if len(e.Tags) > 0 {
			line += "  :" + strings.Join(e.Tags, ":") + ":"
		}

		fmt.Fprintln(io.Out, line)
	}

	if output.Truncated {
		fmt.Fprintf(io.Out, "# %d/%d headings at depth %d\n", output.Headings, output.TotalCount, output.MaxDepth)
		fmt.Fprintf(io.Out, "# → orgx peek %s --max-depth %d  (show children)\n", output.Path, output.MaxDepth+1)
		if output.LastRef != "" {
			fmt.Fprintf(io.Out, "# → orgx get \"%s\"  (expand last heading)\n", output.LastRef)
		}
	}

	return nil
}

func extractRefShort(ref string) string {
	if idx := strings.Index(ref, "::ID:"); idx != -1 {
		return "::ID:" + ref[idx+5:]
	}
	if idx := strings.Index(ref, "::/"); idx != -1 {
		return "::" + ref[idx+2:]
	}
	if idx := strings.Index(ref, "::H:"); idx != -1 {
		return "::H:" + ref[idx+4:]
	}
	return ""
}
