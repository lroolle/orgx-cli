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
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/spf13/cobra"
)

type SetOptions struct {
	IO       *iostreams.IOStreams
	Prompter cmdutil.Prompter

	Ref       string
	Title     string
	Todo      string
	Tags      []string
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

Examples:
  orgx set notes.org::ID:abc --todo DONE
  orgx set notes.org::/Projects --tags +urgent,-old
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
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Set TODO state")
	cmd.Flags().StringSliceVar(&opts.Tags, "tags", nil, "Set tags (+add, -remove)")
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
		return fmt.Errorf("markdown writes not yet supported (span calculation incomplete)")
	}

	content, err := os.ReadFile(ref.Path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	heading, err := shared.FindHeadingFromContent(ref, content)
	if err != nil {
		return err
	}

	newContent, changes, err := applyChanges(string(content), heading, opts)
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

func applyChanges(content string, h *ir.Heading, opts *SetOptions) (string, []string, error) {
	lines := strings.Split(content, "\n")
	var changes []string

	lineIdx := findHeadingLine(lines, h)
	if lineIdx < 0 {
		return "", nil, fmt.Errorf("could not locate heading in file")
	}

	currentLine := lines[lineIdx]

	if opts.Todo != "" && opts.Todo != h.Todo {
		currentLine = replaceTodoInLine(currentLine, h.Todo, opts.Todo)
		changes = append(changes, fmt.Sprintf("TODO: %s → %s", h.Todo, opts.Todo))
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
	return strings.Join(lines, "\n"), changes, nil
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
