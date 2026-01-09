package id

import (
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdID(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "id <command>",
		Short: "Manage heading IDs",
		Long: `Manage heading IDs.

Stable refs (::ID:uuid) are required for safe writes. Use these commands
to add IDs to headings and validate uniqueness (Org :ID: properties, Markdown orgx-id markers).`,
	}

	cmdutil.AddGroup(cmd, "Modify",
		NewCmdEnsure(f, nil),
	)

	cmdutil.AddGroup(cmd, "Query",
		NewCmdList(f, nil),
		NewCmdCheck(f, nil),
	)

	return cmd
}
