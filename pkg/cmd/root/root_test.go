package root

import (
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/cmd/factory"
	"github.com/lroolle/orgx-cli/skills"
)

// The skill is runtime prompt content: an agent drives the commands
// it names. Every user-facing top-level command must appear in the
// embedded SKILL.md, or agents get a contract the binary does not
// keep (and new commands stay invisible to them).
func TestSkillMentionsEveryTopLevelCommand(t *testing.T) {
	cmd := NewCmdRoot(factory.New("test"))
	var missing []string
	for _, c := range cmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" || c.Name() == "version" {
			continue
		}
		if !strings.Contains(skills.OrgxSkill, c.Name()) {
			missing = append(missing, c.Name())
		}
	}
	if len(missing) > 0 {
		t.Fatalf("SKILL.md does not mention: %v — update skills/orgx/SKILL.md alongside the command surface", missing)
	}
}
