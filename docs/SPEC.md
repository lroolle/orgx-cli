# orgx CLI spec (v0)

Goal: a workspace-aware CLI for Org-mode + Markdown, designed for humans and AI agents.

Core properties:
- deterministic outputs
- stable references (refs), never line numbers
- safe edits: patch-based, `--dry-run` first, `--yes` to apply
- stdout is data; stderr is UX

Non-goals (for now):
- full-fidelity round-trip editing of arbitrary Org/Markdown
- perfect Org coverage (we target an explicit subset)
- bidirectional sync between Org and Markdown

---

## CLI conventions

Command shape:

`orgx <domain> <command> [args] [flags]`

Terms:
- path: filesystem path to a document
- ws: workspace name (configured)
- ref: stable selector into a document (heading/node/etc)

Help text:
- help lives with commands (doc generation reads `cobra.Command`)
- command usage syntax follows `gh` conventions: `docs/command-line-syntax.md` (copy the rules)

---

## Global flags and env

Global flags (root persistent):
- `--ws <name>`: select workspace
- `--root <path>`: override root directory (bypasses ws root)
- `--format <auto|org|md|text>`: output format (only for commands that render text)
- `--json`: machine output (see JSON contract)
- `--dry-run`: preview changes without writing
- `--yes`: skip confirmations; required for non-interactive writes

Notes:
- do NOT use `-w` as shorthand for workspace (reserve `-w/--web` for “open in browser” where relevant).
- `--dry-run` MUST NOT write, create backups, or mutate files.

Environment (minimal set; extend only when needed):
- `ORGX_WS`: default workspace (lower precedence than `--ws`)
- `ORGX_ROOT`: default root override (lower precedence than `--root`)
- `ORGX_CONFIG_DIR`: config directory override (else XDG)
- `NO_COLOR`: disable color output
- `PAGER`: default pager

Resolution order (flags win):
- ws: `--ws` > `ORGX_WS` > config `default_workspace` > empty
- root: `--root` > `ORGX_ROOT` > ws.root > `.` (cwd)

---

## Workspace and config

Config file location:
- default: `$XDG_CONFIG_HOME/orgx/config.yaml` (fallback: `$HOME/.config/orgx/config.yaml`)
- override: `ORGX_CONFIG_DIR`

Config schema (v1):

```yaml
version: 1
default_workspace: work
workspaces:
  work:
    root: /abs/path/to/org
    format: org   # org|md|auto (default auto)
    roam_db: /abs/path/to/org-roam.db   # optional
    inbox: /abs/path/to/inbox.org       # optional
```

Workspace subcommands (spec):
- `orgx init`: create config dir + starter config
- `orgx ws add <name> --root <dir> [--format <org|md|auto>] [--roam-db <path>] [--inbox <path>]`
- `orgx ws list`
- `orgx ws show <name>`
- `orgx ws use <name>`: set default workspace
- `orgx ws rm <name>`: remove workspace (refuse if it is default unless `--yes`)

---

## Refs (stable selectors)

Ref input forms:
1) path-only: `<path>`
2) path + selector: `<path>::<selector>`
3) workspace-qualified path: `@<ws>/<path>` or `@<ws>/<path>::<selector>`

Selector grammar:
- Org stable ID: `ID:<uuid>`
- Outline path: `/Outline/Path` (fragile; breaks on rename/move)
- Markdown hash anchor: `H:<hex>` (fragile; changes on content edits)

Parsing rules:
- split on the last `::` occurrence
- `<path>` may contain `:` (Windows drive letters) and is still safe because we split on `::`
- `@<ws>/...` prefix overrides `--ws` and `ORGX_WS`

Ref normalization in JSON output:
- if ws is known and file is under ws root: `@<ws>/<relpath>::ID:<uuid>`
- else: `<abspath>::ID:<uuid>` (or selector form used)

Ref lookup:
- ID refs MUST error if not found
- outline/hash refs are best-effort but still error if they don’t resolve

---

## IR contract (internal, but affects JSON)

We parse Org/Markdown into a shared IR (see `pkg/ir`).

Stability rules:
- `ref` is always the primary identifier for headings/nodes returned
- `span` is best-effort and may shift between versions; never use spans as stable IDs

---

## Output contract

Rule 1: stdout is data.
- in human mode: formatted human output
- in JSON mode: valid JSON only

Rule 2: stderr is UX.
- prompts, progress, warnings, “opening browser…”, and non-data messages

### JSON envelope (v1)

When `--json` is enabled, commands emit:

```json
{
  "version": "v1",
  "workspace": {
    "name": "work",
    "root": "/abs/path"
  },
  "result": {},
  "warnings": [],
  "changes": [],
  "errors": []
}
```

Errors in JSON mode:
- if `--json` is enabled and an error occurs after command execution begins, we MUST still emit a valid JSON envelope with `errors[]` populated and exit non-zero.
- flag/usage errors MAY print usage (stderr) and return non-JSON.

Determinism:
- arrays are sorted (by path/ref/name) before printing JSON
- JSON keys use stable casing and naming (snake_case in JSON tags)

### Error objects

```json
{
  "code": "E_PARSE_FAILED",
  "message": "failed to parse org file",
  "path": "/abs/path/to/file.org",
  "ref": "notes.org::ID:...",
  "details": {}
}
```

### Change objects (write commands)

```json
{
  "path": "/abs/path/to/file.org",
  "kind": "edit",
  "summary": "set TODO to DONE",
  "applied": false,
  "backup_path": ""
}
```

Rules:
- `--dry-run`: `applied=false`, no backup
- apply: `applied=true`, create backup, include `backup_path`

---

## Exit codes

- `0`: success
- `1`: failure (including validation, lookup, parse, write errors)
- `2`: canceled (user declined confirm, or SIGINT while prompting)

---

## Editing model

Edits are patch-based, no surprise mutations:
1) locate target (ref)
2) parse enough structure to compute safe patch
3) compute patch + `changes[]`
4) `--dry-run`: print diff (human) or return diff in JSON change details
5) apply with backup + atomic write

File safety:
- write via temp file + atomic rename where possible
- create backups: `<file>~<timestamp>`
- optional lock file for concurrent writers (future)

---

## Command specs (v0 tree)

Top-level:
- `orgx version`
- `orgx init`
- `orgx ws ...`
- `orgx file ...`
- `orgx heading ...`
- `orgx roam ...` (optional for initial milestone)
- `orgx convert ...` (optional for initial milestone)
- `orgx agenda ...` (later)
- `orgx capture ...` (later)

### `orgx file`

- `orgx file parse <path>`
  - JSON `result`: `ir.Document`
  - human: brief summary + optionally pretty printed outline

- `orgx file outline <path> [--max-depth <n>]`
  - JSON `result`: `{ "path": "...", "headings": [...] }` (minimal heading objects)
  - human: tree view

- `orgx file stats <path>`
  - JSON `result`: counts (headings/tasks/links), sha256, doc_type

- `orgx file search <query> [--in <glob>] [--type <org|md|auto>]`
  - JSON `result`: matches with refs/spans

### `orgx heading`

- `orgx heading list [<path>]`
  - if `<path>` omitted: require `--ws` (list across workspace root)
  - flags: `--level`, `--todo`, `--tag`, `--limit`
  - JSON `result`: list of headings (stable `ref` required)

- `orgx heading view <ref> [--format <org|md|text|auto>]`
  - JSON `result`: heading object (includes body)
  - human: rendered content based on `--format`

- `orgx heading set <ref> [--title <s>] [--todo <s>] [--tags <ops>] [--prop <k=v>] [--scheduled <date>] [--deadline <date>]`
  - requires `--dry-run` or `--yes` in non-interactive mode
  - JSON `result`: updated heading summary + `changes[]`

- `orgx heading move <ref> --to <ref>`
- `orgx heading append <ref> --body <text>`

Tags ops (proposal):
- `--tags +a,+b,-c` (add/remove)

### `orgx roam` (later)

Backed by `org-roam.db` when configured:
- `orgx roam node search <text>`
- `orgx roam node view <node_id>`
- `orgx roam backlinks <node_id>`
- `orgx roam node create --title <s> [--file <path>]`

---

## Open questions (we should decide early)

- JSON flags: keep simple `--json` bool (envelope only), or adopt gh-style `--json <fields>` plus `--jq/--template`.
- workspace-qualified refs: keep as output-only normalization, or support input `@ws/...` (proposed).
- markdown “stable IDs”: frontmatter `id` vs hash anchors; pick one as recommended.

