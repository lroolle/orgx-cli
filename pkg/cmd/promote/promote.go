package promote

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lroolle/orgx-cli/pkg/cmd/heading/shared"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/orgtime"
	"github.com/lroolle/orgx-cli/pkg/textutil"
	"github.com/spf13/cobra"
)

type PromoteOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter
	Prompter cmdutil.Prompter

	Ref       string
	To        string
	Todo      string
	Scheduled string
	Deadline  string
	NoLog     bool
	DryRun    bool
	Confirmed bool
}

type PromoteResult struct {
	FromRef string `json:"from_ref"`
	ToRef   string `json:"to_ref"`
	Title   string `json:"title"`
	OldTodo string `json:"old_todo,omitempty"`
	NewTodo string `json:"new_todo,omitempty"`
}

func NewCmdPromote(f *cmdutil.Factory, runF func(*PromoteOptions) error) *cobra.Command {
	opts := &PromoteOptions{
		IO:       f.IOStreams,
		Prompter: f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "promote <ref>",
		Short: "Move heading to another file with state change",
		Long: `Move a heading from one file to another, optionally changing its state.

State transitions are logged to LOGBOOK drawer by default.
The heading is appended to the end of the target file.

Date formats for --scheduled and --deadline:
  ISO8601:  2026-01-08, 2026-01-08T14:30
  Relative: today, tomorrow, +1d, +2w, -3d
  Org:      <2026-01-08 Thu>

Examples:
  orgx promote INBOX.org::ID:abc --to BACKLOG.org
  orgx promote INBOX.org::ID:abc --to BACKLOG.org --todo TODO
  orgx promote BACKLOG.org::ID:def --to devlog/260108.org --todo STRT
  orgx promote task.org::ID:xyz --to done.org --todo DONE --deadline today`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ref = args[0]

			if runF != nil {
				return runF(opts)
			}
			return promoteRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.To, "to", "", "Target file (required)")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Set new TODO state")
	cmd.Flags().StringVar(&opts.Scheduled, "scheduled", "", "Set SCHEDULED date")
	cmd.Flags().StringVar(&opts.Deadline, "deadline", "", "Set DEADLINE date")
	cmd.Flags().BoolVar(&opts.NoLog, "no-log", false, "Skip LOGBOOK state logging")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview what would happen")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"from_ref", "to_ref", "title", "old_todo", "new_todo"})

	cmd.MarkFlagRequired("to")

	return cmd
}

func promoteRun(opts *PromoteOptions) error {
	ref, err := shared.ParseRefFromArg(opts.Ref)
	if err != nil {
		return err
	}

	if shared.IsMarkdownFile(ref.Path) {
		return fmt.Errorf("promote only supports .org files")
	}

	targetPath := opts.To
	if !filepath.IsAbs(targetPath) {
		cwd, _ := os.Getwd()
		targetPath = filepath.Join(cwd, targetPath)
	}

	if !strings.HasSuffix(strings.ToLower(targetPath), ".org") {
		return fmt.Errorf("promote only supports .org files")
	}

	sourceContent, err := os.ReadFile(ref.Path)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	heading, err := shared.FindHeadingFromContent(ref, sourceContent)
	if err != nil {
		return err
	}

	headingText, remainingContent, err := extractHeading(string(sourceContent), heading)
	if err != nil {
		return err
	}

	cfg := config.LoadOrDefault()
	now := time.Now().In(cfg.GetTimezone())

	newTodo := opts.Todo
	if newTodo == "" {
		newTodo = heading.Todo
	}

	stateChanged := newTodo != heading.Todo
	modifiedHeading := headingText

	if stateChanged {
		modifiedHeading = changeStateInHeading(modifiedHeading, heading.Todo, newTodo)
	}

	if opts.Scheduled != "" || opts.Deadline != "" {
		modifiedHeading, err = addPlanningToHeading(modifiedHeading, heading, opts, now)
		if err != nil {
			return err
		}
	}

	if stateChanged && !opts.NoLog {
		ts := orgtime.Timestamp{Time: now, HasTime: true, Active: false}
		logEntry := orgtime.FormatStateChange(newTodo, heading.Todo, ts)
		modifiedHeading = addLogbookEntry(modifiedHeading, logEntry)
	}

	if stateChanged && cfg.IsDoneState(newTodo) && cfg.ShouldAutoClose() {
		ts := orgtime.Timestamp{Time: now, HasTime: true, Active: false}
		modifiedHeading = addClosedTimestamp(modifiedHeading, ts.String())
	}

	id := heading.Props["ID"]
	newRef := fmt.Sprintf("%s::ID:%s", targetPath, id)

	result := PromoteResult{
		FromRef: opts.Ref,
		ToRef:   newRef,
		Title:   heading.Title,
		OldTodo: heading.Todo,
		NewTodo: newTodo,
	}

	if opts.DryRun {
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, result)
		}
		fmt.Fprintln(opts.IO.Out, "Would move:")
		fmt.Fprintf(opts.IO.Out, "  From: %s\n", opts.Ref)
		fmt.Fprintf(opts.IO.Out, "  To:   %s\n", newRef)
		if stateChanged {
			fmt.Fprintf(opts.IO.Out, "  State: %s -> %s\n", heading.Todo, newTodo)
		}
		fmt.Fprintln(opts.IO.Out)
		fmt.Fprintln(opts.IO.Out, "Modified heading:")
		fmt.Fprintln(opts.IO.Out, modifiedHeading)
		return nil
	}

	if !opts.Confirmed {
		if !opts.IO.CanPrompt() {
			return fmt.Errorf("--yes required in non-interactive mode (no TTY)")
		}
		fmt.Fprintln(opts.IO.ErrOut, "Will move:")
		fmt.Fprintf(opts.IO.ErrOut, "  From: %s\n", opts.Ref)
		fmt.Fprintf(opts.IO.ErrOut, "  To:   %s\n", newRef)
		if stateChanged {
			fmt.Fprintf(opts.IO.ErrOut, "  State: %s -> %s\n", heading.Todo, newTodo)
		}
		if opts.Prompter != nil {
			if err := opts.Prompter.Confirm("Move heading?"); err != nil {
				return cmdutil.CancelError
			}
		}
	}

	var targetContent []byte
	var targetMode os.FileMode = 0644
	lineEnding := "\n"

	if info, err := os.Stat(targetPath); err == nil {
		targetMode = info.Mode()
		targetContent, _ = os.ReadFile(targetPath)
		if len(targetContent) > 0 {
			lineEnding = textutil.DetectLineEnding(string(targetContent))
		}
	}

	var newTargetContent string
	if len(targetContent) == 0 {
		newTargetContent = modifiedHeading + lineEnding
	} else {
		existingContent := string(targetContent)
		if !strings.HasSuffix(existingContent, lineEnding) {
			existingContent += lineEnding
		}
		newTargetContent = existingContent + lineEnding + modifiedHeading + lineEnding
	}

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	if err := os.WriteFile(targetPath, []byte(newTargetContent), targetMode); err != nil {
		return fmt.Errorf("write target file: %w", err)
	}

	sourceInfo, _ := os.Stat(ref.Path)
	sourceMode := sourceInfo.Mode()
	sourceLineEnding := textutil.DetectLineEnding(string(sourceContent))
	newSourceContent := textutil.JoinLines(textutil.SplitLines(remainingContent), sourceLineEnding)

	if err := os.WriteFile(ref.Path, []byte(newSourceContent), sourceMode); err != nil {
		return fmt.Errorf("write source file: %w", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}

	fmt.Fprintf(opts.IO.Out, "Moved: %s -> %s\n", opts.Ref, newRef)
	if stateChanged {
		fmt.Fprintf(opts.IO.Out, "Logged: State %q from %q\n", newTodo, heading.Todo)
	}
	return nil
}

func extractHeading(content string, h *ir.Heading) (string, string, error) {
	lines := textutil.SplitLines(content)
	startIdx := h.Span.Start - 1
	if startIdx < 0 || startIdx >= len(lines) {
		return "", "", fmt.Errorf("could not locate heading")
	}

	headingLevel := h.Level
	endIdx := startIdx + 1

	for endIdx < len(lines) {
		line := lines[endIdx]
		if strings.HasPrefix(line, "*") {
			level := 0
			for _, c := range line {
				if c == '*' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= headingLevel {
				break
			}
		}
		endIdx++
	}

	headingLines := lines[startIdx:endIdx]
	headingText := strings.Join(headingLines, "\n")

	remainingLines := append(lines[:startIdx], lines[endIdx:]...)
	remainingContent := strings.Join(remainingLines, "\n")

	return headingText, remainingContent, nil
}

func changeStateInHeading(heading, oldTodo, newTodo string) string {
	lines := strings.Split(heading, "\n")
	if len(lines) == 0 {
		return heading
	}

	line := lines[0]
	if oldTodo == "" {
		pattern := strings.Index(line, "* ")
		if pattern >= 0 {
			lines[0] = line[:pattern+2] + newTodo + " " + line[pattern+2:]
		}
	} else {
		lines[0] = strings.Replace(line, " "+oldTodo+" ", " "+newTodo+" ", 1)
	}
	return strings.Join(lines, "\n")
}

func addPlanningToHeading(heading string, h *ir.Heading, opts *PromoteOptions, now time.Time) (string, error) {
	lines := strings.Split(heading, "\n")
	if len(lines) == 0 {
		return heading, nil
	}

	scheduled := h.Scheduled
	deadline := h.Deadline

	if opts.Scheduled != "" {
		ts, err := orgtime.Parse(opts.Scheduled, now)
		if err != nil {
			return "", fmt.Errorf("invalid --scheduled date: %w", err)
		}
		scheduled = ts.Format(true)
	}

	if opts.Deadline != "" {
		ts, err := orgtime.Parse(opts.Deadline, now)
		if err != nil {
			return "", fmt.Errorf("invalid --deadline date: %w", err)
		}
		deadline = ts.Format(true)
	}

	var planningParts []string
	if scheduled != "" {
		planningParts = append(planningParts, "SCHEDULED: "+scheduled)
	}
	if deadline != "" {
		planningParts = append(planningParts, "DEADLINE: "+deadline)
	}

	if len(planningParts) == 0 {
		return heading, nil
	}

	planningLine := strings.Join(planningParts, " ")

	insertIdx := 1
	if insertIdx < len(lines) {
		existing := strings.TrimSpace(lines[insertIdx])
		if strings.HasPrefix(existing, "SCHEDULED:") || strings.HasPrefix(existing, "DEADLINE:") || strings.HasPrefix(existing, "CLOSED:") {
			lines[insertIdx] = planningLine
			return strings.Join(lines, "\n"), nil
		}
	}

	newLines := append(lines[:1], append([]string{planningLine}, lines[1:]...)...)
	return strings.Join(newLines, "\n"), nil
}

func addLogbookEntry(heading, entry string) string {
	lines := strings.Split(heading, "\n")

	logbookStart := -1
	logbookEnd := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ":LOGBOOK:" {
			logbookStart = i
		} else if logbookStart >= 0 && trimmed == ":END:" {
			logbookEnd = i
			break
		}
	}

	if logbookStart >= 0 && logbookEnd >= 0 {
		newLines := append(lines[:logbookStart+1], append([]string{entry}, lines[logbookStart+1:]...)...)
		return strings.Join(newLines, "\n")
	}

	insertIdx := 1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "SCHEDULED:") || strings.HasPrefix(line, "DEADLINE:") || strings.HasPrefix(line, "CLOSED:") {
			insertIdx = i + 1
			continue
		}
		if line == ":PROPERTIES:" {
			for j := i; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == ":END:" {
					insertIdx = j + 1
					break
				}
			}
		}
		break
	}

	logbookLines := []string{":LOGBOOK:", entry, ":END:"}
	newLines := append(lines[:insertIdx], append(logbookLines, lines[insertIdx:]...)...)
	return strings.Join(newLines, "\n")
}

func addClosedTimestamp(heading, closed string) string {
	lines := strings.Split(heading, "\n")
	if len(lines) < 2 {
		return heading
	}

	closedLine := "CLOSED: " + closed

	if len(lines) > 1 {
		existing := strings.TrimSpace(lines[1])
		if strings.HasPrefix(existing, "SCHEDULED:") || strings.HasPrefix(existing, "DEADLINE:") {
			lines[1] = closedLine + " " + lines[1]
			return strings.Join(lines, "\n")
		}
		if strings.HasPrefix(existing, "CLOSED:") {
			return heading
		}
	}

	newLines := append(lines[:1], append([]string{closedLine}, lines[1:]...)...)
	return strings.Join(newLines, "\n")
}
