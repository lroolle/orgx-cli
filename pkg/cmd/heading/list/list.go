package list

import (
	"fmt"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmd/heading/shared"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path  string
	Level int
	Todo  string
	Tag   string
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "list <path>",
		Short: "List headings",
		Long:  "List headings from an org or markdown file with optional filters.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Path = args[0]

			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.Level, "level", 0, "Filter by heading level")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Filter by TODO state")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "Filter by tag")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, shared.HeadingFields)

	return cmd
}

type HeadingEntry struct {
	Ref       string   `json:"ref"`
	Level     int      `json:"level"`
	Title     string   `json:"title"`
	Todo      string   `json:"todo,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Scheduled string   `json:"scheduled,omitempty"`
	Deadline  string   `json:"deadline,omitempty"`
}

func listRun(opts *ListOptions) error {
	doc, err := parser.ParseFile(opts.Path)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	headings := collectHeadings(doc.Nodes, opts)

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, headings)
	}

	return printHeadings(opts.IO, headings)
}

func collectHeadings(nodes []ir.Node, opts *ListOptions) []HeadingEntry {
	var entries []HeadingEntry

	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			if matchesFilter(h, opts) {
				entries = append(entries, HeadingEntry{
					Ref:       h.Ref,
					Level:     h.Level,
					Title:     h.Title,
					Todo:      h.Todo,
					Tags:      h.Tags,
					Scheduled: h.Scheduled,
					Deadline:  h.Deadline,
				})
			}

			if len(h.Children) > 0 {
				entries = append(entries, collectHeadings(h.Children, opts)...)
			}
		}
	}

	return entries
}

func matchesFilter(h *ir.Heading, opts *ListOptions) bool {
	if opts.Level > 0 && h.Level != opts.Level {
		return false
	}
	if opts.Todo != "" && h.Todo != opts.Todo {
		return false
	}
	if opts.Tag != "" {
		found := false
		for _, t := range h.Tags {
			if t == opts.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func printHeadings(io *iostreams.IOStreams, headings []HeadingEntry) error {
	for _, h := range headings {
		line := strings.Repeat("*", h.Level)
		if h.Todo != "" {
			line += " " + h.Todo
		}
		line += " " + h.Title
		if len(h.Tags) > 0 {
			line += " :" + strings.Join(h.Tags, ":") + ":"
		}
		fmt.Fprintln(io.Out, line)
	}
	return nil
}
