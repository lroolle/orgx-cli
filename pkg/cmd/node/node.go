// Package node is the roam node surface: org-roam-compatible
// file-level nodes (properties drawer with :ID:, #+title,
// #+filetags) created and listed from the workspace root.
package node

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
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

func NewCmdNode(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node <command>",
		Short: "Roam nodes: org files with a stable ID",
		Long: `A roam node is an org file whose head carries an :ID: property and
a #+title — the same shape org-roam uses, so an existing org-roam
directory is already a valid orgx roam.

The workspace root is the graph: 'orgx ws add main --root ~/org/roam'
once, then node/daily/backlinks all resolve against it.`,
	}
	cmd.AddCommand(NewCmdNodeNew(f, nil))
	cmd.AddCommand(NewCmdNodeList(f, nil))
	return cmd
}

type NewOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter
	Prompter cmdutil.Prompter

	Root      string // resolved roam root
	Title     string
	Tags      []string
	As        string
	DryRun    bool
	Confirmed bool
}

type NewResult struct {
	Path  string `json:"path"`
	ID    string `json:"id"`
	Title string `json:"title"`
}

func NewCmdNodeNew(f *cmdutil.Factory, runF func(*NewOptions) error) *cobra.Command {
	opts := &NewOptions{IO: f.IOStreams, Prompter: f.Prompter}

	cmd := &cobra.Command{
		Use:   "new <title>",
		Short: "Create a roam node",
		Long: `Create a new node file in the workspace root, named the org-roam
way (YYYYMMDDHHMMSS-slug.org), with an :ID: properties drawer,
#+title, and optional #+filetags.

--as records the author (an agent or a person) as an :ORGX_AUTHOR:
property and an @author filetag, so 'orgx find --tag @claude' can
answer "what did the agent write".

Examples:
  orgx node new "SRP protocol notes" --tags auth,apple --yes
  orgx node new "Retro 2026-07" --as claude --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Title = args[0]
			ws, _ := cmd.Flags().GetString("workspace")
			rootFlag, _ := cmd.Flags().GetString("root")
			root, err := roam.ResolveRoot(config.LoadOrDefault(), ws, rootFlag)
			if err != nil {
				return err
			}
			// New pages land in the layout's pages directory; the
			// scan stays vault-wide, so where old nodes live is fine.
			opts.Root = roam.LoadLayout(root).PagesDir(root)
			if runF != nil {
				return runF(opts)
			}
			return newRun(opts)
		},
	}

	cmd.Flags().StringSliceVar(&opts.Tags, "tags", nil, "Filetags for the node")
	cmd.Flags().StringVar(&opts.As, "as", "", "Author (agent or person) recorded on the node")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview what would be created")
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"path", "id", "title"})
	return cmd
}

func newRun(opts *NewOptions) error {
	cfg := config.LoadOrDefault()
	now := time.Now().In(cfg.GetTimezone())

	id := uuid.New().String()
	name := fmt.Sprintf("%s-%s.org", now.Format("20060102150405"), roam.Slug(opts.Title))
	path := filepath.Join(opts.Root, name)

	tags := append([]string(nil), opts.Tags...)
	if opts.As != "" {
		tags = append(tags, "@"+opts.As)
	}

	var b strings.Builder
	b.WriteString(":PROPERTIES:\n")
	fmt.Fprintf(&b, ":ID:       %s\n", id)
	created := orgtime.Timestamp{Time: now, HasTime: true, Active: false}
	fmt.Fprintf(&b, ":CREATED:  %s\n", created.String())
	if opts.As != "" {
		fmt.Fprintf(&b, ":ORGX_AUTHOR: %s\n", opts.As)
	}
	b.WriteString(":END:\n")
	fmt.Fprintf(&b, "#+title: %s\n", opts.Title)
	if len(tags) > 0 {
		fmt.Fprintf(&b, "#+filetags: :%s:\n", strings.Join(tags, ":"))
	}

	result := NewResult{Path: path, ID: id, Title: opts.Title}

	if opts.DryRun {
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, result)
		}
		fmt.Fprintln(opts.IO.Out, "Would create:", path)
		fmt.Fprint(opts.IO.Out, b.String())
		return nil
	}
	if !opts.Confirmed {
		if !opts.IO.CanPrompt() {
			return fmt.Errorf("--yes required in non-interactive mode (no TTY)")
		}
		fmt.Fprintln(opts.IO.ErrOut, "Will create:", path)
		if opts.Prompter != nil {
			if err := opts.Prompter.Confirm("Create node?"); err != nil {
				return cmdutil.CancelError
			}
		}
	}

	if err := os.MkdirAll(opts.Root, 0755); err != nil {
		return fmt.Errorf("create roam root: %w", err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("write node: %w", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, result)
	}
	fmt.Fprintf(opts.IO.Out, "Created: %s (id %s)\n", path, id)
	return nil
}

type ListOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Root   string
	Tag    string
	Search string
	Limit  int
}

func NewCmdNodeList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List roam nodes in the workspace",
		Long: `List every node (org file with a file-level :ID:) under the
workspace root. Files without an ID are counted as skipped, not
silently dropped.

Examples:
  orgx node list --json
  orgx node list --tag @claude          # nodes an agent authored
  orgx node list --search auth --limit 5`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, _ := cmd.Flags().GetString("workspace")
			rootFlag, _ := cmd.Flags().GetString("root")
			root, err := roam.ResolveRoot(config.LoadOrDefault(), ws, rootFlag)
			if err != nil {
				return err
			}
			opts.Root = root
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Tag, "tag", "", "Only nodes carrying this filetag")
	cmd.Flags().StringVar(&opts.Search, "search", "", "Substring match on title")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Max nodes to show (0 = all)")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"path", "id", "title", "tags"})
	return cmd
}

type ListResult struct {
	Root    string          `json:"root"`
	Count   int             `json:"count"`
	Skipped int             `json:"skipped,omitempty"`
	Nodes   []roam.NodeMeta `json:"nodes"`
}

func listRun(opts *ListOptions) error {
	nodes, skipped, err := roam.Scan(opts.Root)
	if err != nil {
		return fmt.Errorf("scan %s: %w", opts.Root, err)
	}

	filtered := nodes[:0]
	for _, n := range nodes {
		if opts.Tag != "" && !hasTag(n.Tags, opts.Tag) {
			continue
		}
		if opts.Search != "" && !strings.Contains(strings.ToLower(n.Title), strings.ToLower(opts.Search)) {
			continue
		}
		filtered = append(filtered, n)
	}
	total := len(filtered)
	truncated := opts.Limit > 0 && total > opts.Limit
	if truncated {
		filtered = filtered[:opts.Limit]
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, ListResult{
			Root: opts.Root, Count: total, Skipped: skipped, Nodes: filtered,
		})
	}

	if total == 0 {
		fmt.Fprintf(opts.IO.Out, "No nodes in %s\n", opts.Root)
		fmt.Fprintln(opts.IO.Out, `# → orgx node new "Title" --yes`)
		return nil
	}
	for _, n := range filtered {
		line := fmt.Sprintf("%s  %q", relOrSelf(opts.Root, n.Path), n.Title)
		if len(n.Tags) > 0 {
			line += "  :" + strings.Join(n.Tags, ":") + ":"
		}
		fmt.Fprintln(opts.IO.Out, line)
	}
	if truncated {
		fmt.Fprintf(opts.IO.Out, "# %d/%d nodes (limit reached) — raise --limit for more\n", opts.Limit, total)
	}
	if skipped > 0 {
		fmt.Fprintf(opts.IO.Out, "# %d org file(s) without a file-level :ID: skipped — orgx id ensure adds one\n", skipped)
	}
	return nil
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

func relOrSelf(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return path
}
