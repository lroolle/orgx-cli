package file

import (
	"github.com/lroolle/orgx-cli/pkg/cmd/file/outline"
	"github.com/lroolle/orgx-cli/pkg/cmd/file/parse"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdFile(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <command>",
		Short: "Work with org and markdown files",
	}

	cmd.AddCommand(parse.NewCmdParse(f, nil))
	cmd.AddCommand(outline.NewCmdOutline(f, nil))

	return cmd
}
