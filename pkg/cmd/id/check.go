package id

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CheckOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path      string
	Recursive bool
}

type IDConflict struct {
	ID        string   `json:"id"`
	Locations []string `json:"locations"`
}

type CheckResult struct {
	Total     int          `json:"total"`
	Unique    int          `json:"unique"`
	Conflicts []IDConflict `json:"conflicts,omitempty"`
}

func NewCmdCheck(f *cmdutil.Factory, runF func(*CheckOptions) error) *cobra.Command {
	opts := &CheckOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "Check for duplicate IDs",
		Long: `Validate that all :ID: properties are unique.

Returns exit code 1 if duplicates are found.

Examples:
  orgx id check notes.org
  orgx id check ~/org --recursive
  orgx id check . -r --json`,
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
			return checkRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Process directories recursively")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"total", "unique", "conflicts"})

	return cmd
}

func checkRun(opts *CheckOptions) error {
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

	idToLocations := make(map[string][]string)
	scanFailed := discoverFailed
	for _, file := range files {
		entries, err := collectIDs(file)
		if err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Warning: %s: %v\n", file, err)
			scanFailed++
			continue
		}
		for _, e := range entries {
			loc := fmt.Sprintf("%s:%d", e.File, e.Line)
			idToLocations[e.ID] = append(idToLocations[e.ID], loc)
		}
	}

	var conflicts []IDConflict
	for id, locs := range idToLocations {
		if len(locs) > 1 {
			slices.Sort(locs)
			conflicts = append(conflicts, IDConflict{
				ID:        id,
				Locations: locs,
			})
		}
	}

	slices.SortFunc(conflicts, func(a, b IDConflict) int {
		return strings.Compare(a.ID, b.ID)
	})

	result := CheckResult{
		Total:     len(idToLocations),
		Unique:    len(idToLocations) - len(conflicts),
		Conflicts: conflicts,
	}

	if opts.Exporter != nil {
		if err := opts.Exporter.Write(opts.IO, result); err != nil {
			return err
		}
		if scanFailed > 0 || len(conflicts) > 0 {
			return cmdutil.SilentError
		}
		return nil
	}

	if len(conflicts) == 0 {
		if scanFailed > 0 {
			return cmdutil.SilentError
		}
		fmt.Fprintf(opts.IO.Out, "OK: %d unique IDs, no duplicates\n", result.Total)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "CONFLICT: %d duplicate IDs found\n\n", len(conflicts))
	for _, c := range conflicts {
		fmt.Fprintf(opts.IO.Out, "  %s:\n", c.ID)
		for _, loc := range c.Locations {
			fmt.Fprintf(opts.IO.Out, "    - %s\n", loc)
		}
	}

	return cmdutil.SilentError
}
