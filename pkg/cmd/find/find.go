package find

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/orgtime"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

type FindOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Query           string
	In              string
	Todo            string
	Tag             string
	After           string
	Before          string
	ScheduledAfter  string
	ScheduledBefore string
	DeadlineAfter   string
	DeadlineBefore  string
	Limit           int

	afterTime           time.Time
	beforeTime          time.Time
	scheduledAfterTime  time.Time
	scheduledBeforeTime time.Time
	deadlineAfterTime   time.Time
	deadlineBeforeTime  time.Time
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

Date formats for filters:
  ISO8601:  2026-01-08, 2026-01-08T14:30
  Relative: today, tomorrow, +1d, +2w, -3d, -7d
  Org:      <2026-01-08 Thu>

Examples:
  orgx find "project" --in ~/org/
  orgx find "" --todo TODO --in ~/org/
  orgx find "" --todo WAIT --in .                    # blocked tasks
  orgx find "" --deadline-before today --in .        # overdue
  orgx find "" --scheduled-after -7d --in .          # scheduled this week
  orgx find "design" --tag work --in ~/org/*.org --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Query = args[0]
			}

			// Default to the roam workspace when one is configured —
			// same rule as backlinks: the workspace root is the graph.
			if opts.In == "" {
				ws, _ := cmd.Flags().GetString("workspace")
				rootFlag, _ := cmd.Flags().GetString("root")
				if root, err := roam.ResolveRoot(config.LoadOrDefault(), ws, rootFlag); err == nil {
					opts.In = root
				} else {
					opts.In = "."
				}
			}

			if err := parseDateFilters(opts); err != nil {
				return err
			}

			if runF != nil {
				return runF(opts)
			}
			return findRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.In, "in", "", "Directory or glob pattern to search")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Filter by TODO state (comma-separated for multiple)")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&opts.After, "after", "", "Items with timestamps after date")
	cmd.Flags().StringVar(&opts.Before, "before", "", "Items with timestamps before date")
	cmd.Flags().StringVar(&opts.ScheduledAfter, "scheduled-after", "", "SCHEDULED after date")
	cmd.Flags().StringVar(&opts.ScheduledBefore, "scheduled-before", "", "SCHEDULED before date")
	cmd.Flags().StringVar(&opts.DeadlineAfter, "deadline-after", "", "DEADLINE after date")
	cmd.Flags().StringVar(&opts.DeadlineBefore, "deadline-before", "", "DEADLINE before date")
	cmd.Flags().IntVar(&opts.Limit, "limit", 50, "Maximum results")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "title", "todo", "tags", "scheduled", "deadline"})

	return cmd
}

func parseDateFilters(opts *FindOptions) error {
	cfg := config.LoadOrDefault()
	now := time.Now().In(cfg.GetTimezone())
	var err error

	if opts.After != "" {
		ts, err := orgtime.Parse(opts.After, now)
		if err != nil {
			return fmt.Errorf("invalid --after date: %w", err)
		}
		opts.afterTime = ts.Time
	}
	if opts.Before != "" {
		ts, err := orgtime.Parse(opts.Before, now)
		if err != nil {
			return fmt.Errorf("invalid --before date: %w", err)
		}
		opts.beforeTime = ts.Time
	}
	if opts.ScheduledAfter != "" {
		ts, err := orgtime.Parse(opts.ScheduledAfter, now)
		if err != nil {
			return fmt.Errorf("invalid --scheduled-after date: %w", err)
		}
		opts.scheduledAfterTime = ts.Time
	}
	if opts.ScheduledBefore != "" {
		ts, err := orgtime.Parse(opts.ScheduledBefore, now)
		if err != nil {
			return fmt.Errorf("invalid --scheduled-before date: %w", err)
		}
		opts.scheduledBeforeTime = ts.Time
	}
	if opts.DeadlineAfter != "" {
		ts, err := orgtime.Parse(opts.DeadlineAfter, now)
		if err != nil {
			return fmt.Errorf("invalid --deadline-after date: %w", err)
		}
		opts.deadlineAfterTime = ts.Time
	}
	if opts.DeadlineBefore != "" {
		ts, err := orgtime.Parse(opts.DeadlineBefore, now)
		if err != nil {
			return fmt.Errorf("invalid --deadline-before date: %w", err)
		}
		opts.deadlineBeforeTime = ts.Time
	}

	return err
}

type FindResult struct {
	Ref       string   `json:"ref"`
	Title     string   `json:"title"`
	Todo      string   `json:"todo,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Scheduled string   `json:"scheduled,omitempty"`
	Deadline  string   `json:"deadline,omitempty"`
	Path      string   `json:"path"`
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
	// A directory means the whole tree — dailies and subtopics live
	// in subdirectories, and a graph search that silently skips them
	// reads as "no matches" when the data is simply one level down.
	if info, err := os.Stat(pattern); err == nil && info.IsDir() {
		var files []string
		walkErr := filepath.Walk(pattern, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".org", ".md":
				files = append(files, path)
			}
			return nil
		})
		return files, walkErr
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
					Ref:       h.Ref,
					Title:     h.Title,
					Todo:      h.Todo,
					Tags:      h.Tags,
					Scheduled: h.Scheduled,
					Deadline:  h.Deadline,
					Path:      extractPath(h.Ref),
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

	if opts.Todo != "" {
		todos := strings.Split(opts.Todo, ",")
		found := false
		for _, t := range todos {
			if strings.TrimSpace(t) == h.Todo {
				found = true
				break
			}
		}
		if !found {
			return false
		}
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

	cfg := config.LoadOrDefault()
	now := time.Now().In(cfg.GetTimezone())

	if !opts.scheduledAfterTime.IsZero() || !opts.scheduledBeforeTime.IsZero() {
		if h.Scheduled == "" {
			return false
		}
		schedTs, err := orgtime.Parse(h.Scheduled, now)
		if err != nil {
			return false
		}
		if !opts.scheduledAfterTime.IsZero() && !schedTs.Time.After(opts.scheduledAfterTime) {
			return false
		}
		if !opts.scheduledBeforeTime.IsZero() && !schedTs.Time.Before(opts.scheduledBeforeTime) {
			return false
		}
	}

	if !opts.deadlineAfterTime.IsZero() || !opts.deadlineBeforeTime.IsZero() {
		if h.Deadline == "" {
			return false
		}
		deadTs, err := orgtime.Parse(h.Deadline, now)
		if err != nil {
			return false
		}
		if !opts.deadlineAfterTime.IsZero() && !deadTs.Time.After(opts.deadlineAfterTime) {
			return false
		}
		if !opts.deadlineBeforeTime.IsZero() && !deadTs.Time.Before(opts.deadlineBeforeTime) {
			return false
		}
	}

	if !opts.afterTime.IsZero() || !opts.beforeTime.IsZero() {
		hasMatch := false
		for _, ts := range []string{h.Scheduled, h.Deadline} {
			if ts == "" {
				continue
			}
			parsed, err := orgtime.Parse(ts, now)
			if err != nil {
				continue
			}
			afterOK := opts.afterTime.IsZero() || parsed.Time.After(opts.afterTime)
			beforeOK := opts.beforeTime.IsZero() || parsed.Time.Before(opts.beforeTime)
			if afterOK && beforeOK {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
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
