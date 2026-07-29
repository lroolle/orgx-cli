---
name: orgx
description: >
  Context-efficient CLI for LLMs to work with Org and Markdown files.
  Minimizes token usage by providing structure-only views, precise section
  retrieval, and surgical edits. Supports GTD workflow with state logging,
  timestamps, and capture/promote operations.
---

# ORGX CLI

**Token-efficient interface for structured documents.**

```
orgx
├── peek <path>          # Structure only (low tokens)
├── get <ref>            # Get specific section
├── find <query>         # Search across files (refs only)
├── set <ref>            # Surgical edit with state logging
├── capture <text>       # Create new heading
├── promote <ref>        # Move heading between files
├── log <ref>            # Show state change history
├── file parse/outline
├── heading list/view/set
└── ws add/list/show/use
```

## Core Workflow

```bash
# 1. Understand structure (cheap - ~100 tokens for 1000-line file)
orgx peek ~/org/projects.org --json

# 2. Find what you need (refs only, not content)
orgx find "design" --in ~/org/ --json

# 3. Get just that section (not whole file)
orgx get ~/org/projects.org::ID:abc123 --json

# 4. Surgical edit (no read needed, auto-logs state change)
orgx set ~/org/projects.org::ID:abc123 --todo DONE --yes
```

## GTD Workflow

```bash
# Capture idea to inbox (uses default_file from config, or INBOX.org in cwd)
orgx capture "Review Gemini API docs" --yes

# Promote to backlog with deadline
orgx promote INBOX.org::ID:abc --to BACKLOG.org --todo TODO --deadline +3d --yes

# Start work (logs state change)
orgx set BACKLOG.org::ID:abc --todo STRT --yes

# Complete (auto-adds CLOSED timestamp)
orgx set BACKLOG.org::ID:abc --todo DONE --yes

# View state history
orgx log BACKLOG.org::ID:abc
```

## Commands

### peek - Structure Only

```bash
orgx peek notes.org
# Output:
# notes.org (1247 lines, 43 headings)
# * TODO Project Alpha  ::ID:abc123  :work:
#   ** NEXT Design      ::/Design
# * DONE Project Beta   ::ID:def456

orgx peek notes.org --json
orgx peek notes.org --max-depth 2
```

### get - Specific Section

```bash
orgx get notes.org::ID:abc123           # org format
orgx get notes.org::ID:abc123 --json    # JSON for agents
orgx get notes.org::/Projects --format md
orgx get notes.org::ID:abc123 --no-children
```

### find - Search Across Files

```bash
orgx find "project" --in ~/org/
orgx find "" --todo TODO --in ~/org/
orgx find "" --todo WAIT,HOLD --in .              # multiple states
orgx find "" --deadline-before today --in .       # overdue
orgx find "" --scheduled-after -7d --in .         # scheduled this week
orgx find "design" --in ~/org/ --json --limit 10
```

Date filters: `--after`, `--before`, `--scheduled-after`, `--scheduled-before`, `--deadline-after`, `--deadline-before`

### set - Surgical Edit

```bash
# Preview first
orgx set notes.org::ID:abc123 --todo DONE --dry-run

# State change (logs to LOGBOOK, auto-closes on DONE/KILL)
orgx set notes.org::ID:abc123 --todo DONE --yes
orgx set notes.org::ID:abc123 --todo STRT --no-log --yes  # skip logging

# Tags
orgx set notes.org::ID:abc123 --tags +urgent,-old --yes

# Timestamps
orgx set notes.org::ID:abc123 --scheduled +3d --yes
orgx set notes.org::ID:abc123 --deadline 2026-01-15 --yes
orgx set notes.org::ID:abc123 --created --yes             # add :CREATED:
```

Date formats: `2026-01-15`, `2026-01-15T14:30`, `today`, `tomorrow`, `+1d`, `+2w`, `-3d`

### capture - Create New Heading

```bash
orgx capture "Review API docs" --yes                       # uses default_file from config
orgx capture "Review API docs" --to INBOX.org --yes
orgx capture "Fix auth bug" --to tasks.org --todo TODO --yes
orgx capture "Submit report" --deadline +3d --tags urgent --yes
```

Creates heading at EOF with auto-generated ID and CREATED timestamp.
If `--to` omitted, uses `[capture] default_file` from config (default: `INBOX.org` in cwd).

### promote - Move Heading Between Files

```bash
orgx promote INBOX.org::ID:abc --to BACKLOG.org --yes
orgx promote INBOX.org::ID:abc --to BACKLOG.org --todo TODO --yes
orgx promote BACKLOG.org::ID:def --to devlog.org --todo STRT --scheduled today --yes
```

Moves heading, logs state change to LOGBOOK, preserves history.

### log - State Change History

```bash
orgx log notes.org::ID:abc123
# Output:
# Project Alpha
# Current: DONE
#
# 2026-01-08 Thu 15:45  DONE <- STRT
# 2026-01-08 Thu 14:30  STRT <- TODO
# 2026-01-07 Wed 09:15  TODO <- IDEA

orgx log notes.org::ID:abc123 --json
orgx log notes.org::ID:abc123 --limit 5
```

## Stable References

| Pattern | Example | Stability |
|---------|---------|-----------|
| `path::ID:uuid` | `notes.org::ID:abc123` | **Stable** |
| `path::/Outline` | `notes.org::/Projects` | Fragile |
| `path::H:hash` | `README.md::H:1a2b` | Fragile |

Always prefer `::ID:` refs from peek/find output.

## Token Efficiency

| Operation | Read Tool | orgx | Savings |
|-----------|-----------|------|---------|
| Understand 1000-line file | 1000 tokens | `peek` → 100 | 90% |
| Find in 5 files | 5000 tokens | `find` → 50 | 99% |
| Get section | 1000 tokens | `get ref` → 100 | 90% |
| Edit heading | 2000 tokens | `set ref` → 0 | 100% |

## Pitfalls

```bash
# Always peek first - don't guess structure
orgx peek notes.org --json        # RIGHT
orgx get notes.org::/Guess        # WRONG - use ref from peek

# Quote refs with special chars
orgx get "notes.org::ID:abc-123"  # RIGHT
orgx get notes.org::ID:abc-123    # May break

# Use --dry-run before writes
orgx set ref --todo DONE --dry-run  # RIGHT
orgx set ref --todo DONE --yes      # After verifying

# Use --json for parsing
orgx find "x" --in ~/org/ --json    # RIGHT - structured
orgx find "x" --in ~/org/           # Human readable

# capture/promote only support .org files
orgx capture "idea" --to notes.org --yes  # RIGHT
orgx capture "idea" --to notes.md --yes   # WRONG
```

## Global Flags

```
--json           Machine output (always use for agents)
--dry-run        Preview changes
--yes            Skip confirmation
--no-log         Skip LOGBOOK state logging (set only)
-w, --workspace  Use specific workspace
```
