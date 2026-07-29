package agentcmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
)

// knownTools is the launch order when --tool is not given.
var knownTools = []string{"claude", "codex"}

func detectTool(explicit string) (string, error) {
	if explicit != "" {
		found := false
		for _, t := range knownTools {
			if t == explicit {
				found = true
			}
		}
		if !found {
			return "", cmdutil.FlagErrorf("unknown --tool %q (claude or codex) — for any other harness, compose 'orgx agent --brief' yourself", explicit)
		}
		if _, err := exec.LookPath(explicit); err != nil {
			return "", cmdutil.WithFix(fmt.Errorf("%s not found in PATH", explicit),
				"install it, or run 'orgx agent --brief' and compose your own harness")
		}
		return explicit, nil
	}
	for _, t := range knownTools {
		if _, err := exec.LookPath(t); err == nil {
			return t, nil
		}
	}
	return "", cmdutil.WithFix(fmt.Errorf("no agent CLI found (tried %s)", strings.Join(knownTools, ", ")),
		"install claude or codex, or run 'orgx agent --brief' and compose your own harness")
}

// invocation builds the child command. claude takes the brief as an
// appended system prompt — the task stays the headline. codex has
// no system-prompt flag, so the brief rides the prompt text.
func invocation(tool, brief, task string) []string {
	switch tool {
	case "claude":
		if task == "" {
			return []string{"claude", "--append-system-prompt", brief}
		}
		return []string{"claude", "--append-system-prompt", brief, "-p", task}
	case "codex":
		if task == "" {
			return []string{"codex", brief + "\n\nAwait the user's direction."}
		}
		return []string{"codex", "exec", brief + "\n\nTask: " + task}
	}
	return nil
}

// printableArgv renders the command for --dry-run with the brief
// elided — it is printed separately, once.
func printableArgv(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		switch {
		case strings.Contains(a, "<orgx-vault>"):
			parts[i] = "<brief>"
		case strings.ContainsAny(a, " \n"):
			parts[i] = fmt.Sprintf("%q", a)
		default:
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}
