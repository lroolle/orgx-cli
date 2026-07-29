package set

import (
	"fmt"
	"os"
	"regexp"
	"slices"
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

type SetOptions struct {
	IO       *iostreams.IOStreams
	Prompter cmdutil.Prompter

	Ref       string
	Title     string
	Todo      string
	Tags      []string
	Scheduled string
	Deadline  string
	Created   bool
	Closed    bool
	NoLog     bool
	DryRun    bool
	Confirmed bool
}

func NewCmdSet(f *cmdutil.Factory, runF func(*SetOptions) error) *cobra.Command {
	opts := &SetOptions{
		IO:       f.IOStreams,
		Prompter: f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "set <ref>",
		Short: "Modify heading by ref",
		Long: `Modify a heading's properties by reference.
No need to read the file first - just specify what to change.

State changes are logged to LOGBOOK drawer by default (use --no-log to skip).
Auto-closes with CLOSED timestamp when transitioning to DONE/KILL states.

Date formats for --scheduled and --deadline:
  ISO8601:  2026-01-08, 2026-01-08T14:30
  Relative: today, tomorrow, +1d, +2w, -3d
  Org:      <2026-01-08 Thu>

Examples:
  orgx set notes.org::ID:abc --todo DONE
  orgx set notes.org::/Projects --tags +urgent,-old
  orgx set notes.org::ID:abc --scheduled +3d --deadline +1w
  orgx set notes.org::ID:abc --todo STRT --no-log
  orgx set notes.org::ID:abc --title "New Title" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ref = args[0]

			if runF != nil {
				return runF(opts)
			}
			return setRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Title, "title", "", "Set title")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Set TODO state (logs to LOGBOOK)")
	cmd.Flags().StringSliceVar(&opts.Tags, "tags", nil, "Set tags (+add, -remove)")
	cmd.Flags().StringVar(&opts.Scheduled, "scheduled", "", "Set SCHEDULED date")
	cmd.Flags().StringVar(&opts.Deadline, "deadline", "", "Set DEADLINE date")
	cmd.Flags().BoolVar(&opts.Created, "created", false, "Add :CREATED: property")
	cmd.Flags().BoolVar(&opts.Closed, "closed", false, "Add CLOSED timestamp")
	cmd.Flags().BoolVar(&opts.NoLog, "no-log", false, "Skip LOGBOOK state logging")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")

	return cmd
}

func setRun(opts *SetOptions) error {
	ref, err := shared.ParseRefFromArg(opts.Ref)
	if err != nil {
		return err
	}

	if shared.IsMarkdownFile(ref.Path) {
		if ref.RefType != shared.RefTypeID {
			return fmt.Errorf("markdown writes require stable ::ID: refs; run: orgx id ensure %s --yes", ref.Path)
		}
		if opts.Todo != "" || len(opts.Tags) > 0 || opts.Scheduled != "" || opts.Deadline != "" || opts.Created || opts.Closed {
			return fmt.Errorf("markdown writes support --title only (no --todo/--tags/--scheduled/--deadline/--created/--closed)")
		}
	}

	content, err := os.ReadFile(ref.Path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	heading, err := shared.FindHeadingFromContent(ref, content)
	if err != nil {
		return err
	}

	var newContent string
	var changes []string
	if shared.IsMarkdownFile(ref.Path) {
		newContent, changes, err = applyMarkdownChanges(string(content), heading, opts)
	} else {
		newContent, changes, err = applyChanges(string(content), heading, opts)
	}
	if err != nil {
		return err
	}

	if len(changes) == 0 {
		fmt.Fprintln(opts.IO.Out, "No changes to apply")
		return nil
	}

	if opts.DryRun {
		fmt.Fprintln(opts.IO.Out, "Changes to apply:")
		for _, c := range changes {
			fmt.Fprintf(opts.IO.Out, "  - %s\n", c)
		}
		fmt.Fprintln(opts.IO.Out)
		fmt.Fprintln(opts.IO.Out, "Use --yes to apply")
		return nil
	}

	if !opts.Confirmed {
		if !opts.IO.CanPrompt() {
			return fmt.Errorf("--yes required in non-interactive mode (no TTY)")
		}
		fmt.Fprintln(opts.IO.ErrOut, "Changes to apply:")
		for _, c := range changes {
			fmt.Fprintf(opts.IO.ErrOut, "  - %s\n", c)
		}
		if opts.Prompter != nil {
			if err := opts.Prompter.Confirm("Apply changes?"); err != nil {
				return cmdutil.CancelError
			}
		}
	}

	info, err := os.Stat(ref.Path)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	mode := info.Mode()

	backupPath := ref.Path + "~" + time.Now().Format("20060102T150405")
	if err := os.WriteFile(backupPath, content, mode); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	if err := os.WriteFile(ref.Path, []byte(newContent), mode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Fprintf(opts.IO.Out, "Updated %s\n", ref.String())

	return nil
}

func applyMarkdownChanges(content string, h *ir.Heading, opts *SetOptions) (string, []string, error) {
	if opts.Title == "" || opts.Title == h.Title {
		return content, nil, nil
	}

	lineEnding := textutil.DetectLineEnding(content)
	lines := textutil.SplitLines(content)
	lineIdx := h.Span.Start - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return "", nil, fmt.Errorf("could not locate heading in file")
	}

	line := lines[lineIdx]
	if ok, updated := replaceMarkdownHeadingTitle(line, opts.Title); ok {
		lines[lineIdx] = updated
		return textutil.JoinLines(lines, lineEnding), []string{fmt.Sprintf("Title: %s -> %s", h.Title, opts.Title)}, nil
	}

	if lineIdx+1 < len(lines) && isSetextUnderline(lines[lineIdx+1]) {
		lines[lineIdx] = opts.Title
		return textutil.JoinLines(lines, lineEnding), []string{fmt.Sprintf("Title: %s -> %s", h.Title, opts.Title)}, nil
	}

	return "", nil, fmt.Errorf("not a markdown heading at line %d", h.Span.Start)
}

var markdownATXHeadingLinePattern = regexp.MustCompile(`^(\s{0,3})(#{1,6})\s+(.+?)(\s+#+\s*)?$`)

func replaceMarkdownHeadingTitle(line, newTitle string) (bool, string) {
	m := markdownATXHeadingLinePattern.FindStringSubmatchIndex(line)
	if m == nil {
		return false, ""
	}

	prefix := line[m[2]:m[3]] + line[m[4]:m[5]]
	suffix := ""
	if m[8] != -1 {
		suffix = line[m[8]:m[9]]
	}
	return true, prefix + " " + newTitle + suffix
}

func isSetextUnderline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r != '=' && r != '-' {
			return false
		}
	}
	return true
}

func applyChanges(content string, h *ir.Heading, opts *SetOptions) (string, []string, error) {
	lineEnding := textutil.DetectLineEnding(content)
	lines := textutil.SplitLines(content)
	var changes []string

	lineIdx := findHeadingLine(lines, h)
	if lineIdx < 0 {
		return "", nil, fmt.Errorf("could not locate heading in file")
	}

	currentLine := lines[lineIdx]
	cfg := config.LoadOrDefault()
	now := time.Now().In(cfg.GetTimezone())

	stateChanged := false
	oldState := h.Todo
	newState := opts.Todo

	if opts.Todo != "" && opts.Todo != h.Todo {
		currentLine = replaceTodoInLine(currentLine, h.Todo, opts.Todo)
		changes = append(changes, fmt.Sprintf("TODO: %s → %s", h.Todo, opts.Todo))
		stateChanged = true
	}

	if opts.Title != "" && opts.Title != h.Title {
		currentLine = replaceTitleInLine(currentLine, h.Title, opts.Title)
		changes = append(changes, fmt.Sprintf("Title: %s → %s", h.Title, opts.Title))
	}

	if len(opts.Tags) > 0 {
		newTags := computeNewTags(h.Tags, opts.Tags)
		if !equalTags(h.Tags, newTags) {
			currentLine = replaceTagsInLine(currentLine, h.Tags, newTags)
			changes = append(changes, fmt.Sprintf("Tags: %v → %v", h.Tags, newTags))
		}
	}

	lines[lineIdx] = currentLine

	insertIdx := lineIdx + 1

	if opts.Scheduled != "" || opts.Deadline != "" || opts.Closed || (stateChanged && cfg.IsDoneState(newState) && cfg.ShouldAutoClose()) {
		planningLine, planningChanges, err := buildPlanningLine(h, opts, now, cfg, stateChanged, newState)
		if err != nil {
			return "", nil, err
		}
		if planningLine != "" {
			existingPlanningIdx := findPlanningLine(lines, insertIdx)
			if existingPlanningIdx >= 0 {
				lines[existingPlanningIdx] = planningLine
			} else {
				lines = insertLine(lines, insertIdx, planningLine)
				insertIdx++
			}
			changes = append(changes, planningChanges...)
		}
	}

	propsInsertIdx := findPropsInsertPoint(lines, lineIdx)
	if propsInsertIdx < 0 {
		propsInsertIdx = insertIdx
	}

	if opts.Created {
		ts := orgtime.Timestamp{Time: now, HasTime: true, Active: false}
		lines, propsInsertIdx = ensureProperty(lines, lineIdx, propsInsertIdx, "CREATED", ts.String())
		changes = append(changes, fmt.Sprintf("CREATED: %s", ts.String()))
	}

	if stateChanged && !opts.NoLog {
		ts := orgtime.Timestamp{Time: now, HasTime: true, Active: false}
		logEntry := orgtime.FormatStateChange(newState, oldState, ts)
		lines, _ = ensureLogbookEntry(lines, lineIdx, logEntry)
		changes = append(changes, fmt.Sprintf("LOGBOOK: State %q from %q", newState, oldState))
	}

	return textutil.JoinLines(lines, lineEnding), changes, nil
}

func buildPlanningLine(h *ir.Heading, opts *SetOptions, now time.Time, cfg *config.Config, stateChanged bool, newState string) (string, []string, error) {
	var parts []string
	var changes []string

	scheduled := h.Scheduled
	deadline := h.Deadline
	closed := h.Closed

	if opts.Scheduled != "" {
		ts, err := orgtime.Parse(opts.Scheduled, now)
		if err != nil {
			return "", nil, fmt.Errorf("invalid --scheduled date: %w", err)
		}
		scheduled = ts.Format(true)
		changes = append(changes, fmt.Sprintf("SCHEDULED: %s", scheduled))
	}

	if opts.Deadline != "" {
		ts, err := orgtime.Parse(opts.Deadline, now)
		if err != nil {
			return "", nil, fmt.Errorf("invalid --deadline date: %w", err)
		}
		deadline = ts.Format(true)
		changes = append(changes, fmt.Sprintf("DEADLINE: %s", deadline))
	}

	if opts.Closed || (stateChanged && cfg.IsDoneState(newState) && cfg.ShouldAutoClose()) {
		ts := orgtime.Timestamp{Time: now, HasTime: true, Active: false}
		closed = ts.String()
		changes = append(changes, fmt.Sprintf("CLOSED: %s", closed))
	}

	if closed != "" {
		parts = append(parts, "CLOSED: "+closed)
	}
	if scheduled != "" {
		parts = append(parts, "SCHEDULED: "+scheduled)
	}
	if deadline != "" {
		parts = append(parts, "DEADLINE: "+deadline)
	}

	if len(parts) == 0 {
		return "", nil, nil
	}
	return strings.Join(parts, " "), changes, nil
}

func findPlanningLine(lines []string, startIdx int) int {
	if startIdx >= len(lines) {
		return -1
	}
	line := strings.TrimSpace(lines[startIdx])
	if strings.HasPrefix(line, "CLOSED:") || strings.HasPrefix(line, "SCHEDULED:") || strings.HasPrefix(line, "DEADLINE:") {
		return startIdx
	}
	return -1
}

func findPropsInsertPoint(lines []string, headingIdx int) int {
	for i := headingIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ":PROPERTIES:") {
			return i
		}
		if strings.HasPrefix(line, "CLOSED:") || strings.HasPrefix(line, "SCHEDULED:") || strings.HasPrefix(line, "DEADLINE:") {
			continue
		}
		break
	}
	return -1
}

func ensureProperty(lines []string, headingIdx, insertIdx int, key, value string) ([]string, int) {
	propsStart := -1
	propsEnd := -1
	for i := headingIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if line == ":PROPERTIES:" {
			propsStart = i
			continue
		}
		if propsStart >= 0 && line == ":END:" {
			propsEnd = i
			break
		}
		if propsStart < 0 && !strings.HasPrefix(line, "CLOSED:") && !strings.HasPrefix(line, "SCHEDULED:") && !strings.HasPrefix(line, "DEADLINE:") {
			break
		}
	}

	propLine := fmt.Sprintf(":%s: %s", key, value)

	if propsStart >= 0 && propsEnd >= 0 {
		for i := propsStart + 1; i < propsEnd; i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), ":"+key+":") {
				lines[i] = propLine
				return lines, insertIdx
			}
		}
		lines = insertLine(lines, propsEnd, propLine)
		return lines, insertIdx + 1
	}

	lines = insertLine(lines, insertIdx, ":END:")
	lines = insertLine(lines, insertIdx, propLine)
	lines = insertLine(lines, insertIdx, ":PROPERTIES:")
	return lines, insertIdx + 3
}

func ensureLogbookEntry(lines []string, headingIdx int, entry string) ([]string, int) {
	logbookStart := -1
	logbookEnd := -1
	searchStart := headingIdx + 1

	for i := searchStart; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "*") {
			break
		}
		if line == ":LOGBOOK:" {
			logbookStart = i
			continue
		}
		if logbookStart >= 0 && line == ":END:" {
			logbookEnd = i
			break
		}
	}

	if logbookStart >= 0 && logbookEnd >= 0 {
		lines = insertLine(lines, logbookStart+1, entry)
		return lines, logbookEnd + 1
	}

	insertIdx := headingIdx + 1
	for i := headingIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "CLOSED:") || strings.HasPrefix(line, "SCHEDULED:") || strings.HasPrefix(line, "DEADLINE:") {
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

	lines = insertLine(lines, insertIdx, ":END:")
	lines = insertLine(lines, insertIdx, entry)
	lines = insertLine(lines, insertIdx, ":LOGBOOK:")
	return lines, insertIdx + 3
}

func insertLine(lines []string, idx int, line string) []string {
	if idx >= len(lines) {
		return append(lines, line)
	}
	lines = append(lines[:idx+1], lines[idx:]...)
	lines[idx] = line
	return lines
}

func findHeadingLine(lines []string, h *ir.Heading) int {
	idx := h.Span.Start - 1 // go-org uses 1-based line numbers
	if idx >= 0 && idx < len(lines) {
		return idx
	}
	return -1
}

func replaceTodoInLine(line, oldTodo, newTodo string) string {
	if oldTodo == "" {
		pattern := regexp.MustCompile(`^(\*+)\s+`)
		return pattern.ReplaceAllString(line, "${1} "+newTodo+" ")
	}
	return strings.Replace(line, " "+oldTodo+" ", " "+newTodo+" ", 1)
}

func replaceTitleInLine(line, oldTitle, newTitle string) string {
	return strings.Replace(line, oldTitle, newTitle, 1)
}

func replaceTagsInLine(line string, oldTags, newTags []string) string {
	if len(oldTags) > 0 {
		oldTagStr := ":" + strings.Join(oldTags, ":") + ":"
		if len(newTags) > 0 {
			newTagStr := ":" + strings.Join(newTags, ":") + ":"
			return strings.Replace(line, oldTagStr, newTagStr, 1)
		}
		return strings.Replace(line, " "+oldTagStr, "", 1)
	}
	if len(newTags) > 0 {
		newTagStr := " :" + strings.Join(newTags, ":") + ":"
		return strings.TrimRight(line, " \t") + newTagStr
	}
	return line
}

func computeNewTags(oldTags []string, tagOps []string) []string {
	tags := make(map[string]bool)
	for _, t := range oldTags {
		tags[t] = true
	}

	for _, op := range tagOps {
		if strings.HasPrefix(op, "+") {
			tags[op[1:]] = true
		} else if strings.HasPrefix(op, "-") {
			delete(tags, op[1:])
		} else {
			tags = make(map[string]bool)
			tags[op] = true
		}
	}

	result := make([]string, 0, len(tags))
	for t := range tags {
		result = append(result, t)
	}
	slices.Sort(result)
	return result
}

func equalTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]bool)
	for _, t := range a {
		am[t] = true
	}
	for _, t := range b {
		if !am[t] {
			return false
		}
	}
	return true
}

