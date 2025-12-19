package root

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmd/file"
	"github.com/lroolle/orgx-cli/pkg/cmd/heading"
	"github.com/lroolle/orgx-cli/pkg/cmd/ws"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRoot(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orgx",
		Short: "Org-mode and Markdown CLI for humans and AI agents",
		Long: `orgx is a workspace-aware CLI that parses and edits Org and Markdown
via a shared structured IR, with Org as the canonical store and
Markdown as a first-class interface format.

Designed for AI agents: deterministic outputs, stable refs, --json.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringP("workspace", "w", "", "Use workspace")
	cmd.PersistentFlags().String("root", "", "Override root directory")
	cmd.PersistentFlags().String("format", "auto", "Output format: org, md, auto")
	cmd.PersistentFlags().Bool("dry-run", false, "Preview changes without writing")
	cmd.PersistentFlags().BoolP("yes", "y", false, "Non-interactive apply")

	cmd.AddCommand(newCmdVersion(f))
	cmd.AddCommand(file.NewCmdFile(f))
	cmd.AddCommand(heading.NewCmdHeading(f))
	cmd.AddCommand(ws.NewCmdWs(f))

	return cmd
}

func newCmdVersion(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(f.IOStreams.Out, "orgx %s\n", f.AppVersion)
		},
	}
}
