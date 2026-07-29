// Package skills embeds the agent skill so the text an agent follows
// always matches the binary it drives — installing from the binary
// cannot drift the way a hand-copied file can.
package skills

import _ "embed"

//go:embed orgx/SKILL.md
var OrgxSkill string

// Name is the skill directory name used on install.
const Name = "orgx"
