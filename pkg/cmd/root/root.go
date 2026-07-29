package root

import (
	"fmt"

	"github.com/lroolle/orgx-cli/pkg/cmd/agentcmd"
	"github.com/lroolle/orgx-cli/pkg/cmd/backlinks"
	"github.com/lroolle/orgx-cli/pkg/cmd/capture"
	"github.com/lroolle/orgx-cli/pkg/cmd/daily"
	"github.com/lroolle/orgx-cli/pkg/cmd/file"
	"github.com/lroolle/orgx-cli/pkg/cmd/find"
	"github.com/lroolle/orgx-cli/pkg/cmd/get"
	"github.com/lroolle/orgx-cli/pkg/cmd/graph"
	"github.com/lroolle/orgx-cli/pkg/cmd/heading"
	"github.com/lroolle/orgx-cli/pkg/cmd/id"
	"github.com/lroolle/orgx-cli/pkg/cmd/links"
	"github.com/lroolle/orgx-cli/pkg/cmd/log"
	"github.com/lroolle/orgx-cli/pkg/cmd/ls"
	"github.com/lroolle/orgx-cli/pkg/cmd/node"
	"github.com/lroolle/orgx-cli/pkg/cmd/peek"
	"github.com/lroolle/orgx-cli/pkg/cmd/promote"
	"github.com/lroolle/orgx-cli/pkg/cmd/serve"
	"github.com/lroolle/orgx-cli/pkg/cmd/set"
	"github.com/lroolle/orgx-cli/pkg/cmd/skillscmd"
	"github.com/lroolle/orgx-cli/pkg/cmd/vaultinit"
	"github.com/lroolle/orgx-cli/pkg/cmd/ws"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRoot(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orgx",
		Short: "Context-efficient CLI for LLMs to work with Org and Markdown",
		Long: `orgx provides a token-efficient interface for AI agents to work with
Org-mode and Markdown files.

Core commands (LLM-optimized):
  peek   Show file structure without loading content
  get    Get specific section by ref
  find   Search headings across files
  set    Modify heading by ref

Use 'orgx <command> --help' for more information.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringP("workspace", "w", "", "Use workspace")
	cmd.PersistentFlags().String("root", "", "Override root directory")
	cmd.PersistentFlags().Bool("dry-run", false, "Preview changes")
	cmd.PersistentFlags().BoolP("yes", "y", false, "Skip confirmation")

	cmdutil.AddGroup(cmd, "Core (LLM-optimized)",
		peek.NewCmdPeek(f, nil),
		get.NewCmdGet(f, nil),
		find.NewCmdFind(f, nil),
		set.NewCmdSet(f, nil),
	)

	cmdutil.AddGroup(cmd, "GTD Workflow",
		capture.NewCmdCapture(f, nil),
		promote.NewCmdPromote(f, nil),
		log.NewCmdLog(f, nil),
	)

	cmdutil.AddGroup(cmd, "Vault",
		vaultinit.NewCmdInit(f, nil),
		node.NewCmdNode(f),
		daily.NewCmdDaily(f, nil),
		graph.NewCmdGraph(f, nil),
		serve.NewCmdServe(f, nil),
	)

	cmdutil.AddGroup(cmd, "Navigation",
		ls.NewCmdLs(f, nil),
		links.NewCmdLinks(f, nil),
		backlinks.NewCmdBacklinks(f, nil),
	)

	cmdutil.AddGroup(cmd, "File operations",
		file.NewCmdFile(f),
	)

	cmdutil.AddGroup(cmd, "Heading operations",
		heading.NewCmdHeading(f),
	)

	cmdutil.AddGroup(cmd, "ID management",
		id.NewCmdID(f),
	)

	cmdutil.AddGroup(cmd, "Workspace",
		ws.NewCmdWs(f),
	)

	cmdutil.AddGroup(cmd, "Agent integration",
		agentcmd.NewCmdAgent(f, nil),
		skillscmd.NewCmdSkills(f),
	)

	cmd.AddCommand(newCmdVersion(f))

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
