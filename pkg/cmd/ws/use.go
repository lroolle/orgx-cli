package ws

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type UseOptions struct {
	IO *iostreams.IOStreams

	Name string
}

func NewCmdUse(f *cmdutil.Factory, runF func(*UseOptions) error) *cobra.Command {
	opts := &UseOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Set default workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			if runF != nil {
				return runF(opts)
			}
			return useRun(opts)
		},
	}

	return cmd
}

func useRun(opts *UseOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.SetDefault(opts.Name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(opts.IO.Out, "Now using workspace: %s\n", opts.Name)

	return nil
}
