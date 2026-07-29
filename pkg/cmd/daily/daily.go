// Package daily is the roam journal: one org file per day under the
// workspace's dailies directory, append-only entries, each stamped
// with its time and — when an agent writes — its author. The daily
// is where agents live in the graph: their work journals into the
// same files a human reads and links from.
package daily

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/orgtime"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/lroolle/orgx-cli/pkg/textutil"
	"github.com/spf13/cobra"
)

type DailyOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter
	Prompter cmdutil.Prompter

	DailiesDir string // resolved
	Text       string
	As         string
	Date       string
	Now        time.Time // resolved clock, in workspace timezone
	DryRun     bool
	Confirmed  bool
}

type DailyResult struct {
	Path    string `json:"path"`
	Date    string `json:"date"`
	Created bool   `json:"created"`         // file was new
	Entry   string `json:"entry,omitempty"` // heading line appended
}

func NewCmdDaily(f *cmdutil.Factory, runF func(*DailyOptions) error) *cobra.Command {
	opts := &DailyOptions{IO: f.IOStreams, Prompter: f.Prompter}

	cmd := &cobra.Command{
		Use:   "daily [text]",
		Short: "Append to (or show) the day's journal",
		Long: `The daily is one org file per day (daily/YYYY-MM-DD.org under the
workspace root), created with an :ID: so it is a first-class node.

With text, appends a timestamped entry heading. --as records the
author as an @author tag on the entry — agents journal their work
into the same file a human reads, and 'orgx find --tag @claude'
answers what the agent did and when. Without text, shows the day.

Examples:
  orgx daily "reviewed the SRP notes" --yes
  orgx daily "reserved 3 aliases for signup flows" --as claude --yes
  orgx daily                       # show today
  orgx daily --date -1d            # show yesterday`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Text = args[0]
			}
			cfg := config.LoadOrDefault()
			ws, _ := cmd.Flags().GetString("workspace")
			rootFlag, _ := cmd.Flags().GetString("root")
			root, err := roam.ResolveRoot(cfg, ws, rootFlag)
			if err != nil {
				return err
			}
			opts.DailiesDir = roam.DailiesDir(cfg, ws, root)
			opts.Now = time.Now().In(cfg.GetTimezone())
			if opts.Date != "" {
				ts, err := orgtime.Parse(opts.Date, opts.Now)
				if err != nil {
					return fmt.Errorf("invalid --date: %w", err)
				}
				// Keep the wall-clock time, move the day.
				y, m, d := ts.Time.Date()
				opts.Now = time.Date(y, m, d, opts.Now.Hour(), opts.Now.Minute(), 0, 0, opts.Now.Location())
			}
			if runF != nil {
				return runF(opts)
			}
			return dailyRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.As, "as", "", "Author (agent or person) tagged on the entry")
	cmd.Flags().StringVar(&opts.Date, "date", "", "Day to target: 2026-07-29, today, -1d")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview what would be appended")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"path", "date", "created", "entry"})
	return cmd
}

func dailyRun(opts *DailyOptions) error {
	date := opts.Now.Format("2006-01-02")
	path := filepath.Join(opts.DailiesDir, date+".org")

	if opts.Text == "" {
		return showDay(opts, path, date)
	}

	entry := fmt.Sprintf("* %s %s", opts.Now.Format("15:04"), opts.Text)
	if opts.As != "" {
		entry += "  :@" + opts.As + ":"
	}

	_, statErr := os.Stat(path)
	creating := os.IsNotExist(statErr)
	result := DailyResult{Path: path, Date: date, Created: creating, Entry: entry}

	if opts.DryRun {
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, result)
		}
		fmt.Fprintf(opts.IO.Out, "Would append to %s:\n%s\n", path, entry)
		return nil
	}
	if !opts.Confirmed {
		if !opts.IO.CanPrompt() {
			return fmt.Errorf("--yes required in non-interactive mode (no TTY)")
		}
		fmt.Fprintf(opts.IO.ErrOut, "Will append to %s:\n%s\n", path, entry)
		if opts.Prompter != nil {
			if err := opts.Prompter.Confirm("Append entry?"); err != nil {
				return cmdutil.CancelError
			}
		}
	}

	if err := os.MkdirAll(opts.DailiesDir, 0755); err != nil {
		return fmt.Errorf("create dailies dir: %w", err)
	}

	var content string
	lineEnding := "\n"
	if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
		content = string(raw)
		lineEnding = textutil.DetectLineEnding(content)
		if !strings.HasSuffix(content, lineEnding) {
			content += lineEnding
		}
	} else {
		content = dailyPreamble(date)
	}
	content += entry + lineEnding

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write daily: %w", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}
	fmt.Fprintf(opts.IO.Out, "Journaled to %s\n", path)
	return nil
}

// dailyPreamble makes a fresh daily a real node: ID drawer + title.
func dailyPreamble(date string) string {
	return ":PROPERTIES:\n:ID:       " + uuid.New().String() +
		"\n:END:\n#+title: " + date + "\n\n"
}

type ShowResult struct {
	Path    string `json:"path"`
	Date    string `json:"date"`
	Exists  bool   `json:"exists"`
	Content string `json:"content,omitempty"`
}

func showDay(opts *DailyOptions, path, date string) error {
	raw, err := os.ReadFile(path)
	exists := err == nil
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, ShowResult{
			Path: path, Date: date, Exists: exists, Content: string(raw),
		})
	}
	if !exists {
		fmt.Fprintf(opts.IO.Out, "No daily for %s yet\n", date)
		fmt.Fprintf(opts.IO.Out, "# → orgx daily \"what happened\" --yes\n")
		return nil
	}
	fmt.Fprint(opts.IO.Out, string(raw))
	return nil
}
