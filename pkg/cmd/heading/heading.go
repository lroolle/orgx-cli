package heading

import (
	"github.com/lroolle/orgx-cli/pkg/cmd/heading/list"
	"github.com/lroolle/orgx-cli/pkg/cmd/heading/set"
	"github.com/lroolle/orgx-cli/pkg/cmd/heading/view"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdHeading(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heading <command>",
		Short: "Manage headings",
	}

	cmdutil.AddGroup(cmd, "Query",
		list.NewCmdList(f, nil),
		view.NewCmdView(f, nil),
	)

	cmdutil.AddGroup(cmd, "Modify",
		set.NewCmdSet(f, nil),
	)

	return cmd
}
