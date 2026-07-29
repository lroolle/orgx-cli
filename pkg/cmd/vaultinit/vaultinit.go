// Package vaultinit scaffolds a vault: the Logseq-inspired default
// layout with an .orgx marker that makes every orgx command inside
// the tree find its graph without configuration.
package vaultinit

import (
	"fmt"
	"os"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

type InitOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Dir string
}

func NewCmdInit(f *cmdutil.Factory, runF func(*InitOptions) error) *cobra.Command {
	opts := &InitOptions{IO: f.IOStreams}

	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Scaffold a vault (journals, pages, whiteboards, assets)",
		Long: `Create the default vault layout in a directory (default: current):

  .orgx/            marker + config — commands inside the tree find
                    the vault by walking up, like git
  journals/         one org file per day (orgx daily)
  pages/            topic nodes (orgx node new)
  whiteboards/      spatial notes (no tooling yet — reserved)
  assets/           images and attachments
  pages/contents.org    the front door
  pages/flashcards.org  always-in-front-of-you facts

The graph itself is NOT a directory: derive it with 'orgx graph'.
Init is idempotent and additive — existing files are never touched,
so running it inside an org-roam directory only adds what's missing.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Dir = args[0]
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				opts.Dir = cwd
			}
			if runF != nil {
				return runF(opts)
			}
			return initRun(opts)
		},
	}
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, []string{"root", "created", "kept"})
	return cmd
}

func initRun(opts *InitOptions) error {
	report, err := roam.InitVault(opts.Dir)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, report)
	}
	fmt.Fprintf(opts.IO.Out, "Vault: %s\n", report.Root)
	for _, c := range report.Created {
		fmt.Fprintf(opts.IO.Out, "  + %s\n", c)
	}
	for _, k := range report.Kept {
		fmt.Fprintf(opts.IO.Out, "  = %s (kept)\n", k)
	}
	fmt.Fprintln(opts.IO.Out, `# → orgx daily "first entry" --yes · orgx node new "First page" --yes`)
	return nil
}
