package ls

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type LsOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path      string
	Recursive bool
	Limit     int
	SortBy    string
}

func NewCmdLs(f *cmdutil.Factory, runF func(*LsOptions) error) *cobra.Command {
	opts := &LsOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "List org/markdown files with stats",
		Long: `List org and markdown files in a directory with line counts and heading counts.

Examples:
  orgx ls                          # Current directory
  orgx ls ~/org/                   # Specific directory
  orgx ls ~/org/ --recursive       # Include subdirectories
  orgx ls ~/org/ --sort headings   # Sort by heading count`,
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
			return lsRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Include subdirectories")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Limit number of files shown")
	cmd.Flags().StringVar(&opts.SortBy, "sort", "name", "Sort by: name, lines, headings")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, defaultFields)

	return cmd
}

var defaultFields = []string{"path", "lines", "headings"}

type FileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Lines    int    `json:"lines"`
	Headings int    `json:"headings"`
	Title    string `json:"title,omitempty"`
}

func lsRun(opts *LsOptions) error {
	files, err := findFiles(opts.Path, opts.Recursive)
	if err != nil {
		return err
	}

	var infos []FileInfo
	for _, file := range files {
		info := getFileInfo(file)
		infos = append(infos, info)
	}

	sortFiles(infos, opts.SortBy)

	if opts.Limit > 0 && len(infos) > opts.Limit {
		infos = infos[:opts.Limit]
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, infos)
	}

	return printLs(opts.IO, infos, opts.Limit > 0 && len(files) > opts.Limit, len(files))
}

func getFileInfo(path string) FileInfo {
	content, err := os.ReadFile(path)
	if err != nil {
		return FileInfo{Path: path, Name: filepath.Base(path)}
	}

	lineCount := strings.Count(string(content), "\n") + 1

	doc, err := parser.ParseFile(path)
	if err != nil {
		return FileInfo{
			Path:  path,
			Name:  filepath.Base(path),
			Lines: lineCount,
		}
	}

	headingCount := countHeadings(doc.Nodes)

	return FileInfo{
		Path:     path,
		Name:     filepath.Base(path),
		Lines:    lineCount,
		Headings: headingCount,
		Title:    doc.Meta.Title,
	}
}

func countHeadings(nodes []ir.Node) int {
	count := 0
	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			count++
			count += countHeadings(h.Children)
		}
	}
	return count
}

func sortFiles(infos []FileInfo, sortBy string) {
	switch sortBy {
	case "lines":
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].Lines > infos[j].Lines
		})
	case "headings":
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].Headings > infos[j].Headings
		})
	default:
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].Name < infos[j].Name
		})
	}
}

func printLs(io *iostreams.IOStreams, infos []FileInfo, truncated bool, total int) error {
	if len(infos) == 0 {
		fmt.Fprintln(io.Out, "No org/markdown files found")
		return nil
	}

	maxName := 0
	for _, info := range infos {
		if len(info.Name) > maxName {
			maxName = len(info.Name)
		}
	}

	for _, info := range infos {
		fmt.Fprintf(io.Out, "%-*s  %5d lines  %3d headings",
			maxName, info.Name, info.Lines, info.Headings)
		if info.Title != "" {
			fmt.Fprintf(io.Out, "  \"%s\"", info.Title)
		}
		fmt.Fprintln(io.Out)
	}

	if truncated {
		fmt.Fprintf(io.Out, "# showing %d/%d files, use --limit 0 for all\n", len(infos), total)
	}

	return nil
}

func findFiles(dir string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".org" || ext == ".md" {
				files = append(files, path)
			}
			return nil
		})
		return files, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".org" || ext == ".md" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}
