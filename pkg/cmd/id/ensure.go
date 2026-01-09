package id

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/lroolle/orgx-cli/pkg/textutil"
	"github.com/spf13/cobra"
)

type EnsureOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path      string
	Recursive bool
	Match     string
	matchRe   *regexp.Regexp // compiled from Match
	Todo      string
	Tag       string
	DryRun    bool
	Confirmed bool
}

type IDChange struct {
	File   string `json:"file"`
	OldRef string `json:"old_ref"`
	NewID  string `json:"new_id"`
	Title  string `json:"title"`
	Line   int    `json:"line"`
}

func NewCmdEnsure(f *cmdutil.Factory, runF func(*EnsureOptions) error) *cobra.Command {
	opts := &EnsureOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "ensure [path]",
		Short: "Add IDs to headings without them",
		Long: `Add IDs to headings that don't have them.

For Org files, this writes an :ID: property in a PROPERTIES drawer.
For Markdown files, this writes an HTML comment marker right after the heading:
  <!-- orgx-id: <uuid> -->

Stable refs (::ID:uuid) are required for safe writes.

Examples:
  orgx id ensure notes.org --dry-run       # Preview changes
  orgx id ensure notes.org --yes           # Apply changes
  orgx id ensure ~/org --recursive --yes   # All files in directory
  orgx id ensure . --todo TODO --yes       # Only TODO headings
  orgx id ensure . --match "Project" --yes # Only matching titles`,
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
			return ensureRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Process directories recursively")
	cmd.Flags().StringVar(&opts.Match, "match", "", "Only headings with title matching pattern")
	cmd.Flags().StringVar(&opts.Todo, "todo", "", "Only headings with specific TODO state")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "Only headings with specific tag")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes without writing")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")

	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"file", "old_ref", "new_id", "title", "line"})

	return cmd
}

func ensureRun(opts *EnsureOptions) error {
	if opts.Match != "" {
		re, err := regexp.Compile(opts.Match)
		if err != nil {
			return fmt.Errorf("invalid --match pattern: %w", err)
		}
		opts.matchRe = re
	}

	path := expandPath(opts.Path)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path: %w", err)
	}

	var files []string
	discoverFailed := 0
	if info.IsDir() {
		errOut := func(format string, args ...interface{}) {
			fmt.Fprintf(opts.IO.ErrOut, format, args...)
		}
		files, discoverFailed = findDocFiles(path, opts.Recursive, errOut)
	} else {
		if !isSupportedDocFile(path) {
			return fmt.Errorf("unsupported file type: %s", path)
		}
		files = []string{path}
	}

	if len(files) == 0 {
		if discoverFailed > 0 {
			if opts.Exporter != nil {
				if err := opts.Exporter.Write(opts.IO, []IDChange{}); err != nil {
					return err
				}
			}
			return cmdutil.SilentError
		}
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, []IDChange{})
		}
		fmt.Fprintln(opts.IO.Out, "No .org/.md files found")
		return nil
	}

	allChanges := []IDChange{}
	scanFailed := discoverFailed
	for _, file := range files {
		changes, err := scanFileForMissingIDs(file, opts)
		if err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Warning: %s: %v\n", file, err)
			scanFailed++
			continue
		}
		allChanges = append(allChanges, changes...)
	}

	if len(allChanges) == 0 {
		if opts.Exporter != nil {
			if err := opts.Exporter.Write(opts.IO, []IDChange{}); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(opts.IO.Out, "All headings already have IDs")
		}
		if scanFailed > 0 {
			return cmdutil.SilentError
		}
		return nil
	}

	// --json with --dry-run: preview mode, output JSON without applying
	if opts.Exporter != nil && opts.DryRun {
		if err := opts.Exporter.Write(opts.IO, allChanges); err != nil {
			return err
		}
		if scanFailed > 0 {
			return cmdutil.SilentError
		}
		return nil
	}

	if opts.DryRun {
		fmt.Fprintf(opts.IO.Out, "Would add IDs to %d headings:\n", len(allChanges))
		for _, c := range allChanges {
			fmt.Fprintf(opts.IO.Out, "  %s → %s\n", c.OldRef, c.NewID)
		}
		fmt.Fprintln(opts.IO.Out)
		fmt.Fprintln(opts.IO.Out, "Use --yes to apply")
		if scanFailed > 0 {
			return cmdutil.SilentError
		}
		return nil
	}

	if !opts.Confirmed {
		if !opts.IO.CanPrompt() {
			return fmt.Errorf("--yes required in non-interactive mode (no TTY)")
		}
		fmt.Fprintf(opts.IO.ErrOut, "Will add IDs to %d headings. Continue? [y/N] ", len(allChanges))
		var response string
		fmt.Fscanln(opts.IO.In, &response)
		if response != "y" && response != "Y" {
			return cmdutil.CancelError
		}
	}

	appliedChanges := []IDChange{}
	writeFailed := 0
	byFile := groupChangesByFile(allChanges)

	// Sort files for deterministic output
	sortedFiles := make([]string, 0, len(byFile))
	for file := range byFile {
		sortedFiles = append(sortedFiles, file)
	}
	slices.Sort(sortedFiles)

	for _, file := range sortedFiles {
		changes := byFile[file]
		if err := applyIDChanges(file, changes); err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Error: %s: %v\n", file, err)
			writeFailed++
			continue
		}
		appliedChanges = append(appliedChanges, changes...)
		if opts.Exporter == nil {
			for _, c := range changes {
				fmt.Fprintf(opts.IO.Out, "Added: %s → ::ID:%s\n", c.OldRef, c.NewID)
			}
		}
	}

	// --json with --yes: output only successfully applied changes
	if opts.Exporter != nil {
		if err := opts.Exporter.Write(opts.IO, appliedChanges); err != nil {
			return err
		}
		if writeFailed > 0 || scanFailed > 0 {
			return cmdutil.SilentError
		}
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "\nAdded %d IDs\n", len(appliedChanges))
	if writeFailed > 0 || scanFailed > 0 {
		return cmdutil.SilentError
	}
	return nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func findOrgFiles(dir string, recursive bool, errOut func(string, ...interface{})) ([]string, int) {
	var files []string
	walkErrors := 0

	if recursive {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				errOut("Warning: %s: %v\n", path, err)
				walkErrors++
				return nil
			}
			if !info.IsDir() && isSupportedDocFile(path) {
				files = append(files, path)
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			errOut("Warning: %s: %v\n", dir, err)
			return nil, 1
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if isSupportedDocFile(path) {
				files = append(files, path)
			}
		}
	}

	return files, walkErrors
}

func findDocFiles(dir string, recursive bool, errOut func(string, ...interface{})) ([]string, int) {
	return findOrgFiles(dir, recursive, errOut)
}

func isSupportedDocFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".org" || ext == ".md" || ext == ".markdown"
}

func scanFileForMissingIDs(path string, opts *EnsureOptions) ([]IDChange, error) {
	doc, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	var changes []IDChange
	scanNodesForMissingIDs(doc.Nodes, path, doc.DocType, opts, &changes)
	return changes, nil
}

func scanNodesForMissingIDs(nodes []ir.Node, path string, docType ir.DocType, opts *EnsureOptions, changes *[]IDChange) {
	for _, n := range nodes {
		h, ok := n.(*ir.Heading)
		if !ok {
			continue
		}

		if _, hasID := h.Props["ID"]; !hasID {
			if matchesFilters(h, opts) {
				newID := uuid.New().String()
				line := h.Span.Start
				if docType == ir.DocTypeMarkdown {
					line = h.Span.End
				}
				*changes = append(*changes, IDChange{
					File:   path,
					OldRef: h.Ref,
					NewID:  newID,
					Title:  h.Title,
					Line:   line,
				})
			}
		}

		if len(h.Children) > 0 {
			scanNodesForMissingIDs(h.Children, path, docType, opts, changes)
		}
	}
}

func matchesFilters(h *ir.Heading, opts *EnsureOptions) bool {
	if opts.matchRe != nil {
		if !opts.matchRe.MatchString(h.Title) {
			return false
		}
	}

	if opts.Todo != "" {
		todos := strings.Split(opts.Todo, ",")
		found := false
		for _, t := range todos {
			if h.Todo == strings.TrimSpace(t) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if opts.Tag != "" {
		tags := strings.Split(opts.Tag, ",")
		found := false
		for _, t := range tags {
			t = strings.TrimSpace(t)
			for _, ht := range h.Tags {
				if ht == t {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func groupChangesByFile(changes []IDChange) map[string][]IDChange {
	result := make(map[string][]IDChange)
	for _, c := range changes {
		result[c.File] = append(result[c.File], c)
	}
	return result
}

func applyIDChanges(path string, changes []IDChange) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".markdown" {
		return applyMarkdownIDChanges(path, changes)
	}
	return applyOrgIDChanges(path, changes)
}

func applyOrgIDChanges(path string, changes []IDChange) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode()

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lineEnding := textutil.DetectLineEnding(string(content))
	lines := textutil.SplitLines(string(content))

	// Apply changes in reverse order to preserve line numbers
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		lineIdx := c.Line - 1 // 1-based to 0-based
		if lineIdx < 0 || lineIdx >= len(lines) {
			continue
		}

		// Check if next line is :PROPERTIES:
		hasDrawer := false
		if lineIdx+1 < len(lines) && strings.TrimSpace(lines[lineIdx+1]) == ":PROPERTIES:" {
			hasDrawer = true
		}

		if hasDrawer {
			// Insert ID into existing drawer (after :PROPERTIES:)
			idLine := fmt.Sprintf(":ID: %s", c.NewID)
			lines = insertLine(lines, lineIdx+2, idLine)
		} else {
			// Create new drawer
			drawer := []string{
				":PROPERTIES:",
				fmt.Sprintf(":ID: %s", c.NewID),
				":END:",
			}
			lines = insertLines(lines, lineIdx+1, drawer)
		}
	}

	return os.WriteFile(path, []byte(textutil.JoinLines(lines, lineEnding)), mode)
}

func applyMarkdownIDChanges(path string, changes []IDChange) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode()

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lineEnding := textutil.DetectLineEnding(string(content))
	lines := textutil.SplitLines(string(content))

	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		lineIdx := c.Line - 1 // 1-based to 0-based
		if lineIdx < 0 || lineIdx >= len(lines) {
			continue
		}

		insertAt := lineIdx + 1
		if insertAt < len(lines) {
			if textutil.OrgxIDMarkerRe.MatchString(lines[insertAt]) {
				continue
			}
		}

		marker := fmt.Sprintf("<!-- orgx-id: %s -->", c.NewID)
		lines = insertLine(lines, insertAt, marker)
	}

	return os.WriteFile(path, []byte(textutil.JoinLines(lines, lineEnding)), mode)
}

func insertLine(lines []string, idx int, line string) []string {
	if idx >= len(lines) {
		return append(lines, line)
	}
	lines = append(lines[:idx+1], lines[idx:]...)
	lines[idx] = line
	return lines
}

func insertLines(lines []string, idx int, newLines []string) []string {
	if idx >= len(lines) {
		return append(lines, newLines...)
	}
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:idx]...)
	result = append(result, newLines...)
	result = append(result, lines[idx:]...)
	return result
}

