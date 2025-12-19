package set

import (
	"fmt"
	"os"
	"regexp"
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
	Exporter cmdutil.Exporter

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
		Short: "Set heading properties",
		Long: `Set properties on a heading.

Examples:
  orgx heading set notes.org::ID:abc --todo DONE
  orgx heading set notes.org::/Title --tags +urgent,-old
  orgx heading set notes.org::ID:abc --title "New Title" --dry-run`,
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
	cmd.Flags().StringSliceVar(&opts.Tags, "tags", nil, "Set tags (+add, -remove, or replace)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes without writing")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")

	return cmd
}

func setRun(opts *SetOptions) error {
	ref, err := shared.ParseRefFromArg(opts.Ref)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(ref.Path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	heading, err := shared.FindHeading(ref)
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
		fmt.Fprintln(opts.IO.Out, "Use --yes to apply changes")
		return nil
	}

	if opts.IO.CanPrompt() && !opts.Confirmed {
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

	backupPath := ref.Path + "~" + time.Now().Format("20060102T150405")
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	if err := os.WriteFile(ref.Path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Fprintf(opts.IO.Out, "Updated %s\n", ref.String())
	fmt.Fprintf(opts.IO.Out, "Backup: %s\n", backupPath)

	return nil
}

func applyChanges(content string, h *ir.Heading, opts *SetOptions) (string, []string, error) {
	lines := strings.Split(content, "\n")
	var changes []string
	newContent := content

	headingLine := findHeadingLine(lines, h)
	if headingLine < 0 {
		return "", nil, fmt.Errorf("could not locate heading in file")
	}

	if opts.Todo != "" && opts.Todo != h.Todo {
		newContent = replaceTodo(newContent, lines[headingLine], h.Todo, opts.Todo)
		changes = append(changes, fmt.Sprintf("TODO: %s -> %s", h.Todo, opts.Todo))
	}

	if opts.Title != "" && opts.Title != h.Title {
		newContent = replaceTitle(newContent, lines[headingLine], h.Title, opts.Title)
		changes = append(changes, fmt.Sprintf("Title: %s -> %s", h.Title, opts.Title))
	}

	if len(opts.Tags) > 0 {
		newTags := computeNewTags(h.Tags, opts.Tags)
		if !equalTags(h.Tags, newTags) {
			newContent = replaceTags(newContent, lines[headingLine], h.Tags, newTags)
			changes = append(changes, fmt.Sprintf("Tags: %v -> %v", h.Tags, newTags))
		}
	}

	return newContent, changes, nil
}

func findHeadingLine(lines []string, h *ir.Heading) int {
	pattern := regexp.MustCompile(`^\*+\s`)
	for i, line := range lines {
		if pattern.MatchString(line) && strings.Contains(line, h.Title) {
			return i
		}
	}
	return -1
}

func replaceTodo(content, oldLine, oldTodo, newTodo string) string {
	var newLine string
	if oldTodo == "" {
		pattern := regexp.MustCompile(`^(\*+)\s+`)
		newLine = pattern.ReplaceAllString(oldLine, "${1} "+newTodo+" ")
	} else {
		newLine = strings.Replace(oldLine, " "+oldTodo+" ", " "+newTodo+" ", 1)
	}
	return strings.Replace(content, oldLine, newLine, 1)
}

func replaceTitle(content, oldLine, oldTitle, newTitle string) string {
	newLine := strings.Replace(oldLine, oldTitle, newTitle, 1)
	return strings.Replace(content, oldLine, newLine, 1)
}

func replaceTags(content, oldLine string, oldTags, newTags []string) string {
	var newLine string

	if len(oldTags) > 0 {
		oldTagStr := ":" + strings.Join(oldTags, ":") + ":"
		if len(newTags) > 0 {
			newTagStr := ":" + strings.Join(newTags, ":") + ":"
			newLine = strings.Replace(oldLine, oldTagStr, newTagStr, 1)
		} else {
			newLine = strings.Replace(oldLine, " "+oldTagStr, "", 1)
		}
	} else if len(newTags) > 0 {
		newTagStr := " :" + strings.Join(newTags, ":") + ":"
		newLine = strings.TrimRight(oldLine, " \t") + newTagStr
	} else {
		newLine = oldLine
	}

	return strings.Replace(content, oldLine, newLine, 1)
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

	var result []string
	for t := range tags {
		result = append(result, t)
	}
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
