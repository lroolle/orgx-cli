// Package skillscmd installs the embedded agent skill: the SKILL.md
// an agent loads is written from the binary itself, so skill text
// and command surface ship together.
package skillscmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/skills"
	"github.com/spf13/cobra"
)

func NewCmdSkills(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills <command>",
		Short: "Agent skills bundled with the binary",
	}
	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdInstall(f))
	return cmd
}

type listResult struct {
	Skills []skillInfo `json:"skills"`
}

type skillInfo struct {
	Name  string `json:"name"`
	Bytes int    `json:"bytes"`
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var exporter cmdutil.Exporter
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List embedded skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result := listResult{Skills: []skillInfo{{Name: skills.Name, Bytes: len(skills.OrgxSkill)}}}
			if exporter != nil {
				return exporter.Write(f.IOStreams, result)
			}
			for _, s := range result.Skills {
				fmt.Fprintf(f.IOStreams.Out, "%s\t%d bytes\n", s.Name, s.Bytes)
			}
			return nil
		},
	}
	cmdutil.AddJSONFlags(cmd, &exporter, []string{"skills"})
	return cmd
}

type installResult struct {
	Path    string `json:"path"`
	Scope   string `json:"scope"`
	Updated bool   `json:"updated"` // an existing file was replaced
}

func newCmdInstall(f *cmdutil.Factory) *cobra.Command {
	var exporter cmdutil.Exporter
	var scope string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the skill for agent runtimes",
		Long: `Write the embedded SKILL.md where agent runtimes discover skills:

  --scope user     ~/.claude/skills/orgx/SKILL.md (default)
  --scope project  ./.claude/skills/orgx/SKILL.md

The skill is embedded at build time, so what installs is exactly
what this binary's commands support.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := scopeDir(scope)
			if err != nil {
				return err
			}
			path := filepath.Join(dir, skills.Name, "SKILL.md")
			_, statErr := os.Stat(path)
			existed := statErr == nil
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("create skill dir: %w", err)
			}
			if err := os.WriteFile(path, []byte(skills.OrgxSkill), 0644); err != nil {
				return fmt.Errorf("write skill: %w", err)
			}
			result := installResult{Path: path, Scope: scope, Updated: existed}
			if exporter != nil {
				return exporter.Write(f.IOStreams, result)
			}
			verb := "Installed"
			if existed {
				verb = "Updated"
			}
			fmt.Fprintf(f.IOStreams.Out, "%s %s\n", verb, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "user", "Install scope: user or project")
	cmdutil.AddJSONFlags(cmd, &exporter, []string{"path", "scope", "updated"})
	return cmd
}

func scopeDir(scope string) (string, error) {
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "skills"), nil
	case "project":
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".claude", "skills"), nil
	}
	return "", cmdutil.WithFix(
		fmt.Errorf("unknown scope %q", scope),
		"use --scope user or --scope project")
}
