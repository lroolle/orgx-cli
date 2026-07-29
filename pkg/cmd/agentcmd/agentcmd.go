// Package agentcmd is the L2 bridge (docs/AGENT-FRAME.md, decision
// 3): orgx does not grow its own head — it hands the vault to the
// agent CLI the user already has. The bridge composes a
// vault-resident brief (layout, standing context, house rules) and
// launches claude or codex inside the vault with it; --brief prints
// the same brief for any other harness to consume.
package agentcmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

type AgentOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Root      string
	Task      string
	Tool      string
	Author    string
	BriefOnly bool
	DryRun    bool
}

type BriefResult struct {
	Root   string `json:"root"`
	Tool   string `json:"tool,omitempty"`
	Author string `json:"author"`
	Brief  string `json:"brief"`
}

func NewCmdAgent(f *cmdutil.Factory, runF func(*AgentOptions) error) *cobra.Command {
	opts := &AgentOptions{IO: f.IOStreams}

	cmd := &cobra.Command{
		Use:   "agent [task]",
		Short: "Hand the vault to your agent CLI (claude, codex)",
		Long: `Launch the agent CLI you already have — claude or codex — inside
the vault, primed with a vault-resident brief: the layout, the
graph's current shape, the standing context from flashcards, recent
journals, and the house rules that keep the graph coherent (link by
id, journal your work, verify after freehand edits).

With a task, runs one-shot and exits; without one, opens an
interactive session in the vault. The agent works with its own
editor and its own permission model — orgx primes it and stays out
of the way.

--brief prints the brief instead of launching anything, so any
other harness can compose it:
  some-agent --system "$(orgx agent --brief)" "garden the vault"

Examples:
  orgx agent                              # interactive, in the vault
  orgx agent "link this week's journals to their project pages"
  orgx agent --tool codex "find and fix broken links"
  orgx agent --brief --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Task = args[0]
			}
			ws, _ := cmd.Flags().GetString("workspace")
			rootFlag, _ := cmd.Flags().GetString("root")
			root, err := roam.ResolveRoot(config.LoadOrDefault(), ws, rootFlag)
			if err != nil {
				return err
			}
			opts.Root = root
			if runF != nil {
				return runF(opts)
			}
			return agentRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Tool, "tool", "", "Agent CLI to launch: claude or codex (default: first found)")
	cmd.Flags().StringVar(&opts.Author, "as", "", "Author the agent journals as (default: the tool name)")
	cmd.Flags().BoolVar(&opts.BriefOnly, "brief", false, "Print the vault brief instead of launching")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would launch, without launching")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"root", "tool", "author", "brief"})
	return cmd
}

func agentRun(opts *AgentOptions) error {
	if opts.Exporter != nil && !opts.BriefOnly {
		return cmdutil.FlagErrorf("--json only applies with --brief; a launched session is not JSON")
	}

	tool := ""
	if !opts.BriefOnly {
		var err error
		if tool, err = detectTool(opts.Tool); err != nil {
			return err
		}
	}
	author := opts.Author
	if author == "" {
		if author = tool; author == "" {
			author = "agent"
		}
	}

	brief, err := buildBrief(opts.Root, author)
	if err != nil {
		return err
	}

	if opts.BriefOnly {
		if opts.Exporter != nil {
			return opts.Exporter.Write(opts.IO, BriefResult{Root: opts.Root, Tool: tool, Author: author, Brief: brief})
		}
		fmt.Fprintln(opts.IO.Out, brief)
		return nil
	}

	argv := invocation(tool, brief, opts.Task)
	if opts.DryRun {
		fmt.Fprintf(opts.IO.Out, "would launch in %s:\n", opts.Root)
		fmt.Fprintf(opts.IO.Out, "  %s\n\n", printableArgv(argv))
		fmt.Fprintln(opts.IO.Out, brief)
		return nil
	}

	// The trace stays honest: say what is being launched, where,
	// and as whom, before the child owns the terminal.
	fmt.Fprintf(opts.IO.ErrOut, "orgx agent → %s · vault %s · journaling as @%s\n", tool, opts.Root, author)

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = opts.Root
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// The child already reported itself on the shared stderr.
			return cmdutil.SilentError
		}
		return fmt.Errorf("launch %s: %w", tool, err)
	}
	return nil
}
