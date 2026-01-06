# orgx CLI

**Context-efficient interface for LLMs to work with Org and Markdown.**

---

## The Problem

LLMs have limited context windows. Every token counts.

```
Old way (wasteful):
  Agent: [Read tool] → 2000-line file → all in context
  Agent: mentally parse, find the 50 lines needed
  Agent: [Edit tool] → still paying for 2000 lines
  Total: ~3000 tokens wasted

New way (efficient):
  Agent: orgx peek file.org → 50 tokens (structure only)
  Agent: orgx get file.org::ID:abc → 100 tokens (just that section)
  Agent: orgx set file.org::ID:abc --todo DONE → 0 tokens (direct edit)
  Total: ~150 tokens, surgical precision
```

---

## Why Org is Perfect for LLMs

Org files are **structured documents**:

```org
* TODO Project Alpha                    ← level, TODO state
:PROPERTIES:
:ID: abc-123                            ← stable ref
:END:
Content here...                         ← body
** NEXT Design                          ← child heading
** TODO Implementation                  ← sibling
```

This structure means:
- **Query by structure** - not grep through text
- **Get exactly what you need** - not everything
- **Edit surgically** - not read-then-write

---

## Core Commands

### peek - Structure Only (Low Tokens)

```bash
orgx peek notes.org
```
```
notes.org (1247 lines, 43 headings)
* TODO Project Alpha      ::ID:abc123  :work:
  ** NEXT Design          ::/Design
  ** TODO Implementation  ::/Implementation
* DONE Project Beta       ::ID:def456  :personal:
```

**Cost:** ~100 tokens for a 1000+ line file.
**Use case:** Understand what's in a file before diving in.

### get - Specific Section

```bash
orgx get notes.org::ID:abc123
```

Returns just that heading and its content. Maybe 50 lines instead of 1247.

**Ref formats:**
- `path::ID:uuid` - **stable**, survives renames
- `path::/Outline/Path` - human readable, fragile
- `path::H:hash` - for markdown

### find - Search Across Files

```bash
orgx find "TODO" --in ~/org/
```
```
notes.org::ID:abc123    "Project Alpha"     :work:
tasks.org::/Weekly      "Review tasks"
inbox.org::ID:xyz       "New idea"          :inbox:
```

Returns **refs only**, not content. Agent picks which to expand.

### set - Surgical Edit

```bash
# No read needed - just specify the change
orgx set notes.org::ID:abc123 --todo DONE

# Add/remove tags
orgx set notes.org::ID:abc123 --tags +urgent,-old

# Preview first
orgx set notes.org::ID:abc123 --todo DONE --dry-run
```

---

## Token Efficiency

| Operation | Read Tool | orgx | Savings |
|-----------|-----------|------|---------|
| Understand 1000-line file | 1000 tokens | `peek` → 100 | 90% |
| Find section in 5 files | 5000 tokens | `find` → 50 | 99% |
| Get specific section | 1000 tokens | `get ref` → 100 | 90% |
| Edit one heading | 2000 tokens | `set ref` → 0 | 100% |

---

## Agent Workflow

```bash
# 1. Understand structure (cheap)
orgx peek ~/org/projects.org --json

# 2. Find what you need
orgx find "design" --in ~/org/ --json

# 3. Get specific section
orgx get ~/org/projects.org::ID:abc123 --json

# 4. Surgical edit
orgx set ~/org/projects.org::ID:abc123 --todo DONE --yes

# 5. Verify (optional)
orgx peek ~/org/projects.org | grep abc123
```

---

## Command Reference

```
orgx
├── peek <path>                    # Structure only
│   --max-depth N                  # Limit heading depth
│   --json
├── get <ref>                      # Get specific section
│   --no-children                  # Heading only
│   --format org|md|text|json
├── find <query>                   # Search across files
│   --in <glob>
│   --todo <state>
│   --tag <tag>
│   --limit N
│   --json
├── set <ref>                      # Modify heading
│   --todo <state>
│   --tags +add,-remove
│   --title "new"
│   --dry-run
│   --yes
├── ls [path]                      # List files with stats
│   --recursive                    # Include subdirs
│   --sort name|lines|headings
│   --limit N
│   --json
├── links <path-or-ref>            # Show outgoing links
│   --kind file|id|http
│   --in <dir>
│   --json
├── backlinks <target>             # Find incoming links
│   --in <dir>
│   --json
├── file
│   ├── parse <path>               # Full IR
│   └── outline <path>             # Alias: peek
├── heading
│   ├── list <path>                # All headings
│   ├── view <ref>                 # Alias: get
│   └── set <ref>                  # Alias: set
├── ws
│   ├── add <name> --root <path>
│   ├── list
│   ├── show [name]
│   └── use <name>
└── version
```

### Global Flags

```
--json           machine output (always use for agents)
--dry-run        preview changes
--yes            skip confirmation
-w, --workspace  use specific workspace
```

---

## Stable References

| Pattern | Example | Stability |
|---------|---------|-----------|
| `path::ID:<uuid>` | `notes.org::ID:abc123` | **Stable** |
| `path::/Outline/Path` | `notes.org::/Projects/CLI` | Fragile |
| `path::H:<hash>` | `README.md::H:1a2b3c` | Fragile |

**Best practice:** Use `:ID:` properties in org files.

```org
* My Heading
:PROPERTIES:
:ID: unique-id-here
:END:
```

---

## JSON Output

All commands support `--json`:

```bash
orgx peek notes.org --json
```

```json
{
  "path": "notes.org",
  "lines": 1247,
  "headings": [
    {
      "ref": "notes.org::ID:abc123",
      "level": 1,
      "title": "Project Alpha",
      "todo": "TODO",
      "tags": ["work"]
    }
  ]
}
```

---

## Smart Hints

When output is truncated, orgx shows continuable hints:

```bash
$ orgx peek large.org --max-depth 1
large.org (2000 lines, 10 headings)
* Introduction  ::ID:abc
* Chapter 1     ::ID:def
...
# 10/47 headings at depth 1
# → orgx peek large.org --max-depth 2  (show children)
# → orgx get "large.org::ID:def"  (expand last heading)
```

```bash
$ orgx find "TODO" --in ~/org/ --limit 10
...
# showing 10 results (limit reached)
# → --limit 20 for more results
# → orgx get "last-ref"  (view last result)
```

---

## Navigation Commands

### ls - Directory Overview

```bash
orgx ls ~/org/
```
```
notes.org    1247 lines  43 headings  "My Notes"
tasks.org     500 lines  20 headings  "Tasks"
```

### links - Outgoing Links

```bash
orgx links notes.org
```
```
[file]   other.org      "Related notes"
[http]   https://...    "Reference"
[id]     abc-123        "Linked heading"
```

### backlinks - Incoming Links

```bash
orgx backlinks notes.org --in ~/org/
```
```
3 backlinks found:
  <- tasks.org::ID:xyz  "Task referencing notes"
  <- index.org::/Links  "Link collection"
```

---

## Tips for Agents

1. **Start with ls** - see what files exist before diving in
2. **Then peek** - understand structure of interesting files
3. **Use refs from output** - don't guess, use what the tool gives you
4. **Follow hints** - truncated output tells you how to continue
5. **Use --json** - structured output is easier to parse
6. **Use links/backlinks** - understand relationships between files
7. **Quote refs with special chars** - `"path::ID:abc-123"`

---

## Architecture

```
pkg/
├── cmd/           # Commands (gh-cli patterns)
│   ├── file/      # parse, outline (peek)
│   ├── heading/   # list, view (get), set
│   └── ws/        # workspace management
├── cmdutil/       # Factory, errors, JSON flags
├── iostreams/     # Testable I/O
├── parser/        # go-org + goldmark wrappers
├── ir/            # Intermediate Representation
└── config/        # Workspace config
```

**Parsers:**
- Org: [niklasfasching/go-org](https://github.com/niklasfasching/go-org)
- Markdown: [yuin/goldmark](https://github.com/yuin/goldmark)

---

## Philosophy

1. **Context is precious** - minimize tokens, maximize value
2. **Structure over text** - org's outline is queryable
3. **Refs over paths** - stable, addressable, portable
4. **Peek before read** - always offer structure-only views
5. **Surgical over wholesale** - patch, don't replace
6. **One call, one purpose** - each command does one thing well

---

## Not Roam

We don't integrate org-roam's sqlite database. Instead:

- **Stable refs:** org's `:ID:` property (built-in)
- **Cross-file search:** `orgx find` with glob patterns
- **Structure awareness:** our parser extracts the hierarchy

org-roam is great for Emacs. orgx is for agents.
