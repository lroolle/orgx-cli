package ws

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type AddOptions struct {
	IO *iostreams.IOStreams

	Name   string
	Root   string
	RoamDB string
	Inbox  string
	Format string
}

func NewCmdAdd(f *cmdutil.Factory, runF func(*AddOptions) error) *cobra.Command {
	opts := &AddOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a workspace",
		Long:  "Add a new workspace with a root directory.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			if opts.Root == "" {
				return cmdutil.FlagErrorf("--root is required")
			}

			if runF != nil {
				return runF(opts)
			}
			return addRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Root, "root", "", "Root directory (required)")
	cmd.Flags().StringVar(&opts.RoamDB, "roam-db", "", "Path to org-roam database")
	cmd.Flags().StringVar(&opts.Inbox, "inbox", "", "Path to inbox file")
	cmd.Flags().StringVar(&opts.Format, "format", "", "Default format: org, md")

	cmd.MarkFlagRequired("root")

	return cmd
}

func addRun(opts *AddOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ws := config.Workspace{
		Root:   opts.Root,
		RoamDB: opts.RoamDB,
		Inbox:  opts.Inbox,
		Format: opts.Format,
	}

	if err := cfg.AddWorkspace(opts.Name, ws); err != nil {
		return err
	}

	if cfg.DefaultWorkspace == "" {
		cfg.DefaultWorkspace = opts.Name
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(opts.IO.Out, "Added workspace: %s\n", opts.Name)
	fmt.Fprintf(opts.IO.Out, "Root: %s\n", opts.Root)

	return nil
}
