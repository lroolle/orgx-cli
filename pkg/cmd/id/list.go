package id

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path      string
	Recursive bool
}

type IDEntry struct {
	ID    string `json:"id"`
	Ref   string `json:"ref"`
	File  string `json:"file"`
	Title string `json:"title"`
	Line  int    `json:"line"`
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "list [path]",
		Short: "List all heading IDs",
		Long: `List all :ID: properties found in org files.

Examples:
  orgx id list notes.org
  orgx id list ~/org --recursive
  orgx id list . -r --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Path = args[0]
			} else {
				opts.Path = "."
			}

			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Process directories recursively")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"id", "ref", "file", "title", "line"})

	return cmd
}

func listRun(opts *ListOptions) error {
	path := expandPath(opts.Path)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path: %w", err)
	}

	var files []string
	if info.IsDir() {
		files, err = findOrgFiles(path, opts.Recursive)
		if err != nil {
			return err
		}
	} else {
		files = []string{path}
	}

	var entries []IDEntry
	for _, file := range files {
		fileEntries, err := collectIDs(file)
		if err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Warning: %s: %v\n", file, err)
			continue
		}
		entries = append(entries, fileEntries...)
	}

	slices.SortFunc(entries, func(a, b IDEntry) int {
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		return a.Line - b.Line
	})

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, entries)
	}

	if len(entries) == 0 {
		fmt.Fprintln(opts.IO.Out, "No IDs found")
		return nil
	}

	for _, e := range entries {
		fmt.Fprintf(opts.IO.Out, "%s  %s  %q\n", e.ID, e.File, e.Title)
	}
	fmt.Fprintf(opts.IO.Out, "\n%d IDs total\n", len(entries))

	return nil
}

func collectIDs(path string) ([]IDEntry, error) {
	doc, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	var entries []IDEntry
	collectIDsFromNodes(doc.Nodes, path, &entries)
	return entries, nil
}

func collectIDsFromNodes(nodes []ir.Node, path string, entries *[]IDEntry) {
	for _, n := range nodes {
		h, ok := n.(*ir.Heading)
		if !ok {
			continue
		}

		if id, hasID := h.Props["ID"]; hasID {
			*entries = append(*entries, IDEntry{
				ID:    id,
				Ref:   h.Ref,
				File:  path,
				Title: h.Title,
				Line:  h.Span.Start,
			})
		}

		if len(h.Children) > 0 {
			collectIDsFromNodes(h.Children, path, entries)
		}
	}
}
