---
name: orgx
description: >
  Context-efficient CLI for LLMs to work with Org and Markdown files.
  Minimizes token usage by providing structure-only views, precise section
  retrieval, and surgical edits. Use this skill when working with org files,
  markdown files, or hybrid vaults.
---

# ORGX CLI

**Token-efficient interface for structured documents.**

```
orgx
├── peek <path>          # Structure only (low tokens)
├── get <ref>            # Get specific section
├── find <query>         # Search across files (refs only)
├── set <ref>            # Surgical edit
├── file parse/outline   # Full parsing
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

# 4. Surgical edit (no read needed)
orgx set ~/org/projects.org::ID:abc123 --todo DONE --yes
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
orgx find "" --tag work --in ~/org/*.org
orgx find "design" --in ~/org/ --json --limit 10
```

### set - Surgical Edit

```bash
# Preview first
orgx set notes.org::ID:abc123 --todo DONE --dry-run

# Apply
orgx set notes.org::ID:abc123 --todo DONE --yes
orgx set notes.org::ID:abc123 --tags +urgent,-old --yes
orgx set notes.org::ID:abc123 --title "New Title" --yes
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
```

## Global Flags

```
--json           Machine output (always use for agents)
--dry-run        Preview changes
--yes            Skip confirmation
-w, --workspace  Use specific workspace
```
