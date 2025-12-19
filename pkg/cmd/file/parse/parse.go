package parse

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type ParseOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path string
}

func NewCmdParse(f *cmdutil.Factory, runF func(*ParseOptions) error) *cobra.Command {
	opts := &ParseOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "parse <path>",
		Short: "Parse file to IR",
		Long:  "Parse an org or markdown file and output the intermediate representation.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Path = args[0]

			if runF != nil {
				return runF(opts)
			}
			return parseRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.Exporter, defaultFields)

	return cmd
}

var defaultFields = []string{
	"path",
	"sha256",
	"doc_type",
	"meta",
	"nodes",
}

func parseRun(opts *ParseOptions) error {
	doc, err := parser.ParseFile(opts.Path)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, doc)
	}

	return printHumanReadable(opts.IO, doc)
}

func printHumanReadable(io *iostreams.IOStreams, doc *ir.Document) error {
	fmt.Fprintf(io.Out, "File: %s\n", doc.Path)
	fmt.Fprintf(io.Out, "Type: %s\n", doc.DocType)
	if doc.Meta.Title != "" {
		fmt.Fprintf(io.Out, "Title: %s\n", doc.Meta.Title)
	}
	fmt.Fprintf(io.Out, "Headings: %d\n", countHeadings(doc.Nodes))
	return nil
}

func countHeadings(nodes []ir.Node) int {
	count := 0
	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			count++
			count += countHeadings(h.Children)
		}
	}
	return count
}
