package links

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
	"github.com/spf13/cobra"
)

type LinksOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Path   string
	Ref    string
	Kind   string
	InDir  string
}

func NewCmdLinks(f *cmdutil.Factory, runF func(*LinksOptions) error) *cobra.Command {
	opts := &LinksOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "links <path-or-ref>",
		Short: "Show outgoing links from a file or section",
		Long: `Extract and display links from an org/markdown file or specific section.

Examples:
  orgx links notes.org                    # All links in file
  orgx links notes.org::ID:abc123         # Links from specific section
  orgx links notes.org --kind file        # Only file links
  orgx links --in ~/org/                  # All links in directory`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if strings.Contains(args[0], "::") {
					opts.Ref = args[0]
					opts.Path = extractPath(args[0])
				} else {
					opts.Path = args[0]
				}
			}

			if opts.Path == "" && opts.InDir == "" {
				return fmt.Errorf("provide a path or use --in")
			}

			if runF != nil {
				return runF(opts)
			}
			return linksRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Kind, "kind", "", "Filter by link kind: file, id, http")
	cmd.Flags().StringVar(&opts.InDir, "in", "", "Search in directory")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, defaultFields)

	return cmd
}

var defaultFields = []string{"target", "kind", "desc"}

type LinkOutput struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
	Desc   string `json:"desc,omitempty"`
}

func linksRun(opts *LinksOptions) error {
	var files []string

	if opts.InDir != "" {
		var err error
		files, err = findFiles(opts.InDir)
		if err != nil {
			return err
		}
	} else {
		files = []string{opts.Path}
	}

	var allLinks []LinkOutput

	for _, file := range files {
		doc, err := parser.ParseFile(file)
		if err != nil {
			continue
		}

		links := collectLinks(doc.Nodes, opts.Ref, file)
		for _, l := range links {
			if opts.Kind != "" && string(l.Kind) != opts.Kind {
				continue
			}
			allLinks = append(allLinks, LinkOutput{
				Source: l.Source,
				Target: l.Target,
				Kind:   string(l.Kind),
				Desc:   l.Desc,
			})
		}
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, allLinks)
	}

	return printLinks(opts.IO, allLinks)
}

type linkWithSource struct {
	Source string
	Target string
	Kind   ir.LinkKind
	Desc   string
}

func collectLinks(nodes []ir.Node, refFilter, file string) []linkWithSource {
	var links []linkWithSource

	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			if refFilter != "" && h.Ref != refFilter {
				links = append(links, collectLinks(h.Children, refFilter, file)...)
				continue
			}

			for _, l := range h.Links {
				links = append(links, linkWithSource{
					Source: h.Ref,
					Target: l.Target,
					Kind:   l.Kind,
					Desc:   l.Desc,
				})
			}

			if refFilter == "" {
				links = append(links, collectLinks(h.Children, refFilter, file)...)
			}
		}
	}

	return links
}

func printLinks(io *iostreams.IOStreams, links []LinkOutput) error {
	if len(links) == 0 {
		fmt.Fprintln(io.Out, "No links found")
		return nil
	}

	for _, l := range links {
		kindStr := fmt.Sprintf("[%s]", l.Kind)
		line := fmt.Sprintf("%-8s %s", kindStr, l.Target)
		if l.Desc != "" {
			line += fmt.Sprintf("  \"%s\"", l.Desc)
		}
		fmt.Fprintln(io.Out, line)
	}

	return nil
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

func extractPath(ref string) string {
	if idx := strings.Index(ref, "::"); idx != -1 {
		return ref[:idx]
	}
	return ref
}
