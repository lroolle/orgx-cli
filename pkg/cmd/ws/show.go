package ws

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ShowOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Name string
}

func NewCmdShow(f *cmdutil.Factory, runF func(*ShowOptions) error) *cobra.Command {
	opts := &ShowOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show workspace details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Name = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return showRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"name", "root", "roam_db", "inbox", "format"})

	return cmd
}

func showRun(opts *ShowOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ws, err := cfg.GetWorkspace(opts.Name)
	if err != nil {
		return err
	}

	name := opts.Name
	if name == "" {
		name = cfg.DefaultWorkspace
	}

	entry := WorkspaceEntry{
		Name:      name,
		Root:      ws.Root,
		RoamDB:    ws.RoamDB,
		Inbox:     ws.Inbox,
		Format:    ws.Format,
		IsDefault: name == cfg.DefaultWorkspace,
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, entry)
	}

	fmt.Fprintf(opts.IO.Out, "Name: %s\n", entry.Name)
	fmt.Fprintf(opts.IO.Out, "Root: %s\n", entry.Root)
	if entry.RoamDB != "" {
		fmt.Fprintf(opts.IO.Out, "Roam DB: %s\n", entry.RoamDB)
	}
	if entry.Inbox != "" {
		fmt.Fprintf(opts.IO.Out, "Inbox: %s\n", entry.Inbox)
	}
	if entry.Format != "" {
		fmt.Fprintf(opts.IO.Out, "Format: %s\n", entry.Format)
	}
	if entry.IsDefault {
		fmt.Fprintln(opts.IO.Out, "(default)")
	}

	return nil
}
