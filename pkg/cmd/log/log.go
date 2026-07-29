package log

import (
	"fmt"
	"os"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmd/heading/shared"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/spf13/cobra"
)

type LogOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Ref   string
	Limit int
}

type LogResult struct {
	Ref     string            `json:"ref"`
	Title   string            `json:"title"`
	Todo    string            `json:"todo,omitempty"`
	Entries []ir.StateChange  `json:"entries"`
}

func NewCmdLog(f *cmdutil.Factory, runF func(*LogOptions) error) *cobra.Command {
	opts := &LogOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "log <ref>",
		Short: "Show state change history for a heading",
		Long: `Display the LOGBOOK state change history for a heading.

Shows all state transitions logged in the LOGBOOK drawer, newest first.

Examples:
  orgx log notes.org::ID:abc
  orgx log notes.org::ID:abc --json
  orgx log notes.org::ID:abc --limit 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ref = args[0]

			if runF != nil {
				return runF(opts)
			}
			return logRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Maximum entries to show (0 = all)")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "title", "todo", "entries"})

	return cmd
}

func logRun(opts *LogOptions) error {
	ref, err := shared.ParseRefFromArg(opts.Ref)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(ref.Path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	heading, err := shared.FindHeadingFromContent(ref, content)
	if err != nil {
		return err
	}

	entries := heading.Logbook
	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
	}

	result := LogResult{
		Ref:     opts.Ref,
		Title:   heading.Title,
		Todo:    heading.Todo,
		Entries: entries,
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}

	if len(entries) == 0 {
		fmt.Fprintln(opts.IO.Out, "No state change history")
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "%s\n", heading.Title)
	if heading.Todo != "" {
		fmt.Fprintf(opts.IO.Out, "Current: %s\n", heading.Todo)
	}
	fmt.Fprintln(opts.IO.Out)

	for _, e := range entries {
		from := e.OldState
		if from == "" {
			from = "(created)"
		}
		ts := strings.Trim(e.Timestamp, "[]")
		fmt.Fprintf(opts.IO.Out, "%s  %s <- %s\n", ts, e.NewState, from)
	}

	if opts.Limit > 0 && len(heading.Logbook) > opts.Limit {
		fmt.Fprintf(opts.IO.Out, "\n# showing %d of %d entries\n", opts.Limit, len(heading.Logbook))
		fmt.Fprintf(opts.IO.Out, "# → --limit %d for more\n", opts.Limit*2)
	}

	return nil
}
