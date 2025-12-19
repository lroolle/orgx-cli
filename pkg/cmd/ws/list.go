package ws

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"name", "root", "roam_db", "inbox", "format"})

	return cmd
}

type WorkspaceEntry struct {
	Name      string `json:"name"`
	Root      string `json:"root"`
	RoamDB    string `json:"roam_db,omitempty"`
	Inbox     string `json:"inbox,omitempty"`
	Format    string `json:"format,omitempty"`
	IsDefault bool   `json:"is_default"`
}

func listRun(opts *ListOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Workspaces) == 0 {
		fmt.Fprintln(opts.IO.Out, "No workspaces configured. Use 'orgx ws add' to add one.")
		return nil
	}

	var entries []WorkspaceEntry
	for name, ws := range cfg.Workspaces {
		entries = append(entries, WorkspaceEntry{
			Name:      name,
			Root:      ws.Root,
			RoamDB:    ws.RoamDB,
			Inbox:     ws.Inbox,
			Format:    ws.Format,
			IsDefault: name == cfg.DefaultWorkspace,
		})
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, entries)
	}

	for _, e := range entries {
		marker := "  "
		if e.IsDefault {
			marker = "* "
		}
		fmt.Fprintf(opts.IO.Out, "%s%s\t%s\n", marker, e.Name, e.Root)
	}

	return nil
}
