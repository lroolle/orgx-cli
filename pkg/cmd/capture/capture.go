package capture

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
	"github.com/lroolle/orgx-cli/pkg/textutil"
	"github.com/spf13/cobra"
)

type CaptureOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter
	Prompter cmdutil.Prompter

	Text      string
	To        string
	Todo      string
	Tags      []string
	Scheduled string
	Deadline  string
	DryRun    bool
	Confirmed bool
}

type CaptureResult struct {
	Ref   string `json:"ref"`
	Path  string `json:"path"`
	Title string `json:"title"`
	Todo  string `json:"todo,omitempty"`
	ID    string `json:"id"`
}

func NewCmdCapture(f *cmdutil.Factory, runF func(*CaptureOptions) error) *cobra.Command {
	opts := &CaptureOptions{
		IO:       f.IOStreams,
		Prompter: f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "capture <text>",
		Short: "Create new heading at end of file",
		Long: `Capture a new idea or task by creating a heading at the end of a file.

Creates a new heading with:
- Auto-generated unique ID
- CREATED timestamp in properties
- Optional TODO state, tags, and scheduling

If --to is omitted, uses [capture] default_file from config (default: INBOX.org in cwd).
Supports ~/path expansion for home directory.

Date formats for --scheduled and --deadline:
  ISO8601:  2026-01-08, 2026-01-08T14:30
  Relative: today, tomorrow, +1d, +2w, -3d
  Org:      <2026-01-08 Thu>

Examples:
  orgx capture "Review Gemini API docs"                    # uses default file
  orgx capture "Review Gemini API docs" --to INBOX.org
  orgx capture "Fix auth bug" --to tasks.org --todo TODO
  orgx capture "Weekly review" --todo LOOP --scheduled +1w
  orgx capture "Submit report" --deadline +3d --tags urgent,report`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Text = args[0]

			if runF != nil {
				return runF(opts)
			}
			return captureRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.To, "to", "", "Target file (default: from config or INBOX.org)")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Set TODO state (default: IDEA from config)")
	cmd.Flags().StringSliceVar(&opts.Tags, "tags", nil, "Add tags")
	cmd.Flags().StringVar(&opts.Scheduled, "scheduled", "", "Set SCHEDULED date")
	cmd.Flags().StringVar(&opts.Deadline, "deadline", "", "Set DEADLINE date")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview what would be created")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "path", "title", "todo", "id"})

	return cmd
}

func captureRun(opts *CaptureOptions) error {
	cfg := config.LoadOrDefault()
	now := time.Now().In(cfg.GetTimezone())

	targetPath := opts.To
	if targetPath == "" {
		targetPath = cfg.GetCaptureFile()
	}
	if !filepath.IsAbs(targetPath) {
		if strings.HasPrefix(targetPath, "~/") {
			home, _ := os.UserHomeDir()
			targetPath = filepath.Join(home, targetPath[2:])
		} else {
			cwd, _ := os.Getwd()
			targetPath = filepath.Join(cwd, targetPath)
		}
	}

	if !strings.HasSuffix(strings.ToLower(targetPath), ".org") {
		return fmt.Errorf("capture only supports .org files")
	}

	todo := opts.Todo
	if todo == "" {
		todo = cfg.GetCaptureState()
	}

	id := uuid.New().String()
	createdTs := orgtime.Timestamp{Time: now, HasTime: true, Active: false}

	var headingLines []string
	headingLine := fmt.Sprintf("* %s %s", todo, opts.Text)
	if len(opts.Tags) > 0 {
		headingLine += " :" + strings.Join(opts.Tags, ":") + ":"
	}
	headingLines = append(headingLines, headingLine)

	if opts.Scheduled != "" || opts.Deadline != "" {
		planningParts := []string{}
		if opts.Scheduled != "" {
			ts, err := orgtime.Parse(opts.Scheduled, now)
			if err != nil {
				return fmt.Errorf("invalid --scheduled date: %w", err)
			}
			planningParts = append(planningParts, "SCHEDULED: "+ts.Format(true))
		}
		if opts.Deadline != "" {
			ts, err := orgtime.Parse(opts.Deadline, now)
			if err != nil {
				return fmt.Errorf("invalid --deadline date: %w", err)
			}
			planningParts = append(planningParts, "DEADLINE: "+ts.Format(true))
		}
		headingLines = append(headingLines, strings.Join(planningParts, " "))
	}

	headingLines = append(headingLines, ":PROPERTIES:")
	headingLines = append(headingLines, fmt.Sprintf(":ID: %s", id))
	headingLines = append(headingLines, fmt.Sprintf(":CREATED: %s", createdTs.String()))
	headingLines = append(headingLines, ":END:")

	newHeading := strings.Join(headingLines, "\n")

	result := CaptureResult{
		Ref:   fmt.Sprintf("%s::ID:%s", targetPath, id),
		Path:  targetPath,
		Title: opts.Text,
		Todo:  todo,
		ID:    id,
	}

	if opts.DryRun {
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, result)
		}
		fmt.Fprintln(opts.IO.Out, "Would create:")
		fmt.Fprintln(opts.IO.Out, newHeading)
		fmt.Fprintf(opts.IO.Out, "\nIn: %s\n", targetPath)
		return nil
	}

	if !opts.Confirmed {
		if !opts.IO.CanPrompt() {
			return fmt.Errorf("--yes required in non-interactive mode (no TTY)")
		}
		fmt.Fprintln(opts.IO.ErrOut, "Will create:")
		fmt.Fprintln(opts.IO.ErrOut, newHeading)
		fmt.Fprintf(opts.IO.ErrOut, "\nIn: %s\n", targetPath)
		if opts.Prompter != nil {
			if err := opts.Prompter.Confirm("Create heading?"); err != nil {
				return cmdutil.CancelError
			}
		}
	}

	var content []byte
	var mode os.FileMode = 0644
	lineEnding := "\n"

	if info, err := os.Stat(targetPath); err == nil {
		mode = info.Mode()
		content, _ = os.ReadFile(targetPath)
		if len(content) > 0 {
			lineEnding = textutil.DetectLineEnding(string(content))
		}
	}

	var newContent string
	if len(content) == 0 {
		newContent = newHeading + lineEnding
	} else {
		existingContent := string(content)
		if !strings.HasSuffix(existingContent, lineEnding) {
			existingContent += lineEnding
		}
		newContent = existingContent + lineEnding + newHeading + lineEnding
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(targetPath, []byte(newContent), mode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}

	fmt.Fprintf(opts.IO.Out, "Created: %s\n", result.Ref)
	return nil
}
