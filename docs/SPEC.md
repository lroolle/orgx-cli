# orgx CLI spec (v0)

orgx is a small CLI to make large Org files cheap for AI agents:
- peek directory structure (tree)
- peek document structure (toc)
- fetch an exact section by ref (slice)
- search across many files and return refs (search)
- apply a few safe, structured edits (later)

We are not building “Org-mode in Go”, and we do not need Org-roam DB integration to hit the core goal.

---

## Why this exists (agent constraints)

Agents (Codex CLI / Claude Code style) have hard limits:
- finite context window (can’t hold a whole vault + code + plan)
- tool calls cost time; repeated `sed`/`rg`/`cat` burns tokens and latency
- rewriting whole files is risky (format loss, accidental deletes)

So orgx must:
- return structure first, content second
- support output size limits everywhere (`--max-*`, `--limit`, `--max-depth`)
- make it easy to re-fetch the *same* section later (stable refs)
- keep stdout machine-clean; send UX noise to stderr

---

## Principles

- stdout is data; stderr is UX
  - data = text slice or JSON
  - UX = warnings, prompts, progress, diff previews
- deterministic ordering
  - stable sort order for lists (path, then ref)
- no implicit writes
  - write commands require `--yes` or an interactive confirm (TTY only)
  - `--dry-run` never writes, never creates backups

---

## Workspace and config (optional)

orgx should work without any config:
- pass `--root` (directory) and explicit paths

Config location:
- default: `$XDG_CONFIG_HOME/orgx/config.yaml` (fallback: `$HOME/.config/orgx/config.yaml`)
- override: `ORGX_CONFIG_DIR`

Config schema (v1):

```yaml
version: 1
default_workspace: work
workspaces:
  work:
    root: /abs/path/to/org
```

Resolution order:
- workspace: `--ws` > `ORGX_WS` > config `default_workspace`
- root: `--root` > `ORGX_ROOT` > ws.root > `.`

Workspace commands (v0):
- `orgx init`
- `orgx ws add <name> --root <dir>`
- `orgx ws list`
- `orgx ws show <name>`
- `orgx ws use <name>`
- `orgx ws rm <name> [--yes]`

---

## Refs (stable selectors)

Ref forms:
1) `<path>`
2) `<path>::<selector>`
3) `@<ws>/<path>` or `@<ws>/<path>::<selector>`

Selector forms (v0):
- `ID:<value>` (preferred)
- `/Outline/Path` (fragile)

Notes:
- split on the last `::` so Windows paths remain valid
- `@<ws>/...` overrides `--ws`

Stability rules:
- if a heading has `:ID:`, its ref uses `ID:...`
- if no ID exists, commands may fall back to outline refs; agents should consider them unstable

---

## Output model

Global output flags:
- `--json`: JSON output (stable, command-specific schemas)
- `--format <org|text>`: text rendering mode (only for text outputs)

Output size controls (required for “agent-friendly”):
- list commands: `--limit`, `--max-depth`
- slice/search commands: `--max-bytes`, `--max-lines`, `--context`
- `--full` disables truncation (danger: huge output)

Rule: in `--json` mode, stdout MUST be valid JSON only.

---

## Command set (v0)

Minimal. Everything else is later.

Top-level:
- `orgx version`
- `orgx init` (optional)
- `orgx ws ...` (optional)
- `orgx file ...`
- `orgx heading ...`
- `orgx search ...`

### `orgx file tree`

Peek directory structure.

`orgx file tree [<root>] [--max-depth <n>] [--type org|all] [--limit <n>]`

JSON schema (v0):

```json
{
  "root": "/abs/root",
  "entries": [
    { "path": "notes.org", "kind": "file" },
    { "path": "projects/", "kind": "dir" }
  ]
}
```

Rules:
- `path` is relative to `root`
- deterministic ordering

### `orgx file toc`

Peek document structure (headings only).

`orgx file toc <path> [--max-depth <n>] [--match <re>] [--todo <kw>] [--tag <tag>] [--with-id] [--limit <n>]`

JSON schema (v0):

```json
{
  "path": "/abs/path/to/file.org",
  "headings": [
    {
      "ref": "file.org::ID:abc",
      "level": 2,
      "title": "Build orgx",
      "todo": "TODO",
      "tags": ["cli", "org"]
    }
  ]
}
```

### `orgx heading view`

Fetch an exact subtree/section by ref.

`orgx heading view <ref> [--format org|text] [--full] [--max-bytes <n>] [--max-lines <n>]`

Text mode:
- prints the (possibly truncated) subtree

JSON schema (v0):

```json
{
  "ref": "file.org::ID:abc",
  "path": "/abs/path/to/file.org",
  "subtree": "* TODO Title...\n..."
}
```

### `orgx search`

Search across a directory/workspace and return refs + small excerpts.

`orgx search <query> [--in <root>] [--type org|all] [--context <n>] [--limit <n>]`

JSON schema (v0):

```json
{
  "root": "/abs/root",
  "query": "deadline",
  "matches": [
    {
      "path": "notes.org",
      "ref": "notes.org::ID:abc",
      "excerpt": "SCHEDULED: <2025-12-19 Fri> DEADLINE: <...>"
    }
  ]
}
```

Search rules:
- best-effort mapping from match location -> nearest heading ref
- if no heading found, `ref` may be empty

---

## Later (explicitly out of v0)

- Org-roam DB integration (SQLite)
  - we can get 80% by scanning `:ID:` and `[[id:...]]` links and using workspace search for backlinks
- agenda/capture
- full Markdown support
- free-form “edit anything” commands

