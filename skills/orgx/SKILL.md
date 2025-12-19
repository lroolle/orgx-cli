---
name: orgx
description: >
  Org-mode and Markdown CLI for workspace operations. Use this skill when
  working with org files, markdown files, org-roam nodes, or hybrid vaults.
  Triggers on: org parsing, heading CRUD, vault search, agenda queries,
  md↔org conversion.
---

# ORGX CLI

```
orgx
├── init                              # Setup config
├── ws (workspace)
│   ├── add <name> --root <dir>       # --roam-db
│   ├── list
│   ├── show <name>
│   └── use <name>
├── file
│   ├── parse <path>                  # --json
│   ├── outline <path>                # --max-depth
│   ├── search <query>                # --in <glob>
│   └── stats <path>
├── heading
│   ├── list <path|ws>                # --level --todo --tag --json
│   ├── view <ref>                    # --format org|md|text
│   ├── set <ref>                     # --title --todo --tags --prop --dry-run
│   ├── move <ref> --to <ref>
│   └── append <ref> --body <text>
├── roam
│   ├── node search "<text>"          # --json
│   ├── node view <node_id>
│   ├── backlinks <node_id>           # --json
│   └── node create --title "..."
├── capture
│   ├── add --title "..." --body "..."
│   └── templates list|show
├── agenda
│   ├── list                          # --from --to --json
│   ├── next                          # --limit
│   └── overdue
├── convert
│   ├── md2org <path.md> --to <target>
│   └── org2md <ref> --to <path.md>
└── export
    └── md|html <ref> --to <path>
```

Global: `--ws`, `--root`, `--json`, `--format`, `--dry-run`, `--yes`

## Stable References

Agents use refs, not line numbers:

| Pattern | Example | Stability |
|---------|---------|-----------|
| `path::ID:<uuid>` | `notes.org::ID:8f3c...` | Stable |
| `path::/Outline/Path` | `notes.org::/Projects/CLI` | Fragile |
| `path::H:<hash>` | `README.md::H:a1b2...` | Fragile |

## Parse Files

```bash
orgx file parse notes.org --json
orgx file parse README.md --json
orgx file outline project.org --max-depth 2
```

## Heading Operations

```bash
# List headings
orgx heading list notes.org --level 2 --todo TODO --json

# View specific heading
orgx heading view notes.org::ID:8f3c... --format org

# Edit heading (always use --dry-run first)
orgx heading set notes.org::ID:8f3c... --todo NEXT --tags +ai,+wip --dry-run
orgx heading set notes.org::ID:8f3c... --todo NEXT --tags +ai,+wip --yes
```

## Workspace Management

```bash
orgx ws add work --root ~/org --roam-db ~/.emacs.d/org-roam.db
orgx ws use work
orgx ws list
```

## Conversion

```bash
# Import markdown to org inbox
orgx convert md2org draft.md --to ~/org/inbox.org --json

# Export org heading to markdown
orgx convert org2md notes.org::ID:8f3c... --to output.md
```

## Capture (Inbox)

```bash
orgx capture add --title "New idea" --body "..." --template inbox
```

## Agenda

```bash
orgx agenda list --from today --to +7d --json
orgx agenda next --limit 10
orgx agenda overdue
```

## JSON Output

All `--json` outputs follow stable schema:

```json
{
  "version": "v1",
  "workspace": "work",
  "result": { ... },
  "warnings": [],
  "changes": [],
  "errors": []
}
```

## Error Codes

| Code | Meaning |
|------|---------|
| `E_NO_WORKSPACE` | No workspace configured |
| `E_NOT_FOUND_REF` | Reference not found |
| `E_PARSE_FAILED` | Parse error |
| `E_CONFLICT_LOCKED` | File locked by another process |
| `E_WRITE_FORBIDDEN` | Write not allowed |

## Pitfalls

```bash
# Always use --dry-run before writes
orgx heading set ... --dry-run    # RIGHT - preview first
orgx heading set ... --yes        # After verifying dry-run

# Refs with special chars need quoting
orgx heading view "notes.org::ID:8f3c-a1b2"   # RIGHT
orgx heading view notes.org::ID:8f3c-a1b2     # May break

# Prefer ID refs over outline paths (stability)
notes.org::ID:8f3c...        # Stable
notes.org::/Projects/CLI     # Changes on rename
```
