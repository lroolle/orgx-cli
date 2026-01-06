package find

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type FindOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Query string
	In    string
	Todo  string
	Tag   string
	Limit int
}

func NewCmdFind(f *cmdutil.Factory, runF func(*FindOptions) error) *cobra.Command {
	opts := &FindOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Search headings across files",
		Long: `Search for headings matching criteria across files.
Returns refs only, not content - pick which to expand with 'get'.

Examples:
  orgx find "project" --in ~/org/
  orgx find "" --todo TODO --in ~/org/
  orgx find "design" --tag work --in ~/org/*.org --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Query = args[0]
			}

			if opts.In == "" {
				opts.In = "."
			}

			if runF != nil {
				return runF(opts)
			}
			return findRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.In, "in", "", "Directory or glob pattern to search")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Filter by TODO state")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "Filter by tag")
	cmd.Flags().IntVar(&opts.Limit, "limit", 50, "Maximum results")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "title", "todo", "tags"})

	return cmd
}

type FindResult struct {
	Ref   string   `json:"ref"`
	Title string   `json:"title"`
	Todo  string   `json:"todo,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	Path  string   `json:"path"`
}

func findRun(opts *FindOptions) error {
	files, err := findFiles(opts.In)
	if err != nil {
		return err
	}

	var results []FindResult
	truncated := false

	for _, file := range files {
		doc, err := parser.ParseFile(file)
		if err != nil {
			continue
		}

		matches := searchHeadings(doc.Nodes, opts)
		results = append(results, matches...)

		if opts.Limit > 0 && len(results) >= opts.Limit {
			truncated = len(results) > opts.Limit
			results = results[:opts.Limit]
			break
		}
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, results)
	}

	return printResults(opts.IO, results, truncated, opts)
}

func findFiles(pattern string) ([]string, error) {
	info, err := os.Stat(pattern)
	if err == nil && info.IsDir() {
		pattern = filepath.Join(pattern, "*.org")
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		mdPattern := strings.TrimSuffix(pattern, ".org") + ".md"
		if mdPattern != pattern {
			mdMatches, _ := filepath.Glob(mdPattern)
			matches = append(matches, mdMatches...)
		}
	}

	return matches, nil
}

func searchHeadings(nodes []ir.Node, opts *FindOptions) []FindResult {
	var results []FindResult

	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			if matchesQuery(h, opts) {
				results = append(results, FindResult{
					Ref:   h.Ref,
					Title: h.Title,
					Todo:  h.Todo,
					Tags:  h.Tags,
					Path:  extractPath(h.Ref),
				})
			}

			if len(h.Children) > 0 {
				results = append(results, searchHeadings(h.Children, opts)...)
			}
		}
	}

	return results
}

func matchesQuery(h *ir.Heading, opts *FindOptions) bool {
	if opts.Query != "" {
		if !strings.Contains(strings.ToLower(h.Title), strings.ToLower(opts.Query)) {
			return false
		}
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

func extractPath(ref string) string {
	if idx := strings.Index(ref, "::"); idx != -1 {
		return ref[:idx]
	}
	return ref
}

func printResults(io *iostreams.IOStreams, results []FindResult, truncated bool, opts *FindOptions) error {
	if len(results) == 0 {
		fmt.Fprintln(io.Out, "No matches found")
		return nil
	}

	for _, r := range results {
		line := r.Ref
		line += fmt.Sprintf("\t\"%s\"", r.Title)
		if r.Todo != "" {
			line += "\t" + r.Todo
		}
		if len(r.Tags) > 0 {
			line += "\t:" + strings.Join(r.Tags, ":") + ":"
		}
		fmt.Fprintln(io.Out, line)
	}

	if truncated {
		fmt.Fprintf(io.Out, "# showing %d results (limit reached)\n", opts.Limit)
		fmt.Fprintf(io.Out, "# → --limit %d for more results\n", opts.Limit*2)
		if len(results) > 0 {
			lastRef := results[len(results)-1].Ref
			fmt.Fprintf(io.Out, "# → orgx get \"%s\"  (view last result)\n", lastRef)
		}
	}

	return nil
}
