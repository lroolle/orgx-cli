package ws

import (
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdWs(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ws <command>",
		Aliases: []string{"workspace"},
		Short:   "Manage workspaces",
	}

	cmd.AddCommand(NewCmdAdd(f, nil))
	cmd.AddCommand(NewCmdList(f, nil))
	cmd.AddCommand(NewCmdShow(f, nil))
	cmd.AddCommand(NewCmdUse(f, nil))

	return cmd
}
