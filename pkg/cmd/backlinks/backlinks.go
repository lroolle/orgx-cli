package backlinks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

type BacklinksOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Target string
	InDir  string
}

func NewCmdBacklinks(f *cmdutil.Factory, runF func(*BacklinksOptions) error) *cobra.Command {
	opts := &BacklinksOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "backlinks <target>",
		Short: "Find links pointing to a file or ID",
		Long: `Find all links in a directory that point to the specified target.

Target can be:
  - A file path: finds [[file:target.org]] links
  - An ID: finds [[id:uuid]] links

Examples:
  orgx backlinks notes.org --in ~/org/
  orgx backlinks abc-123 --in ~/org/              # Find ID links
  orgx backlinks projects/main.org --in ~/org/`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Target = args[0]

			// Default to the roam workspace when one is configured —
			// backlinks are a graph question, and the workspace root
			// is the graph. No workspace, no --in: current directory.
			if opts.InDir == "" {
				ws, _ := cmd.Flags().GetString("workspace")
				rootFlag, _ := cmd.Flags().GetString("root")
				if root, err := roam.ResolveRoot(config.LoadOrDefault(), ws, rootFlag); err == nil {
					opts.InDir = root
				} else {
					opts.InDir = "."
				}
			}

			if runF != nil {
				return runF(opts)
			}
			return backlinksRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.InDir, "in", "", "Directory to search (default: roam workspace, else cwd)")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, defaultFields)

	return cmd
}

var defaultFields = []string{"source", "target", "title"}

type BacklinkOutput struct {
	Source      string `json:"source"`
	SourceTitle string `json:"source_title"`
	Target      string `json:"target"`
	Desc        string `json:"desc,omitempty"`
}

func backlinksRun(opts *BacklinksOptions) error {
	files, err := findFiles(opts.InDir)
	if err != nil {
		return err
	}

	var backlinks []BacklinkOutput

	for _, file := range files {
		doc, err := parser.ParseFile(file)
		if err != nil {
			continue
		}

		matches := findBacklinks(doc.Nodes, opts.Target, file)
		backlinks = append(backlinks, matches...)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, backlinks)
	}

	return printBacklinks(opts.IO, backlinks)
}

func findBacklinks(nodes []ir.Node, target, file string) []BacklinkOutput {
	var backlinks []BacklinkOutput

	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			for _, l := range h.Links {
				if matchesTarget(l, target, file) {
					backlinks = append(backlinks, BacklinkOutput{
						Source:      h.Ref,
						SourceTitle: h.Title,
						Target:      l.Target,
						Desc:        l.Desc,
					})
				}
			}

			backlinks = append(backlinks, findBacklinks(h.Children, target, file)...)
		}
	}

	return backlinks
}

func matchesTarget(l *ir.Link, target, sourceFile string) bool {
	switch l.Kind {
	case ir.LinkKindID:
		return l.Target == target || strings.HasSuffix(l.Target, target)
	case ir.LinkKindFile:
		linkTarget := l.Target
		if strings.HasPrefix(linkTarget, "file:") {
			linkTarget = strings.TrimPrefix(linkTarget, "file:")
		}

		if linkTarget == target {
			return true
		}

		if strings.Contains(linkTarget, "::") {
			linkTarget = strings.Split(linkTarget, "::")[0]
		}

		targetBase := filepath.Base(target)
		linkBase := filepath.Base(linkTarget)
		if targetBase == linkBase {
			return true
		}

		if strings.HasSuffix(linkTarget, target) || strings.HasSuffix(target, linkTarget) {
			return true
		}
	}

	return false
}

func printBacklinks(io *iostreams.IOStreams, backlinks []BacklinkOutput) error {
	if len(backlinks) == 0 {
		fmt.Fprintln(io.Out, "No backlinks found")
		return nil
	}

	fmt.Fprintf(io.Out, "%d backlinks found:\n", len(backlinks))
	for _, bl := range backlinks {
		fmt.Fprintf(io.Out, "  <- %s  \"%s\"\n", extractRefShort(bl.Source), bl.SourceTitle)
	}

	return nil
}

func extractRefShort(ref string) string {
	if idx := strings.Index(ref, "::ID:"); idx != -1 {
		parts := strings.SplitN(ref, "::", 2)
		return filepath.Base(parts[0]) + "::" + parts[1]
	}
	if idx := strings.Index(ref, "::/"); idx != -1 {
		parts := strings.SplitN(ref, "::", 2)
		return filepath.Base(parts[0]) + "::" + parts[1]
	}
	return ref
}

func findFiles(dir string) ([]string, error) {
	var files []string
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
