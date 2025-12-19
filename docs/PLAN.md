# orgx implementation plan (checklists)

Principles:
- keep commands small; use Options struct + `runF` injection
- stdout is data; stderr is UX
- table-driven tests for parsing/formatting
- read-only first; edits later

---

## Milestones

### M0: CLI foundation

- [ ] Root command owns: global flags, usage/help overrides, exit code mapping
- [ ] `iostreams` supports: TTY detection, `CanPrompt()`, color on/off, pager (later)
- [ ] `cmdutil` supports: FlagError, SilentError, CancelError, group helper
- [ ] JSON output: stable envelope v1, deterministic ordering
- [ ] Config loader: XDG path + `ORGX_CONFIG_DIR`
- [ ] Workspace resolver: `--ws/ORGX_WS/default_workspace`, `--root/ORGX_ROOT/ws.root`

Definition of done:
- [ ] `orgx version` works
- [ ] `orgx help` output is stable
- [ ] basic unit tests for error typing and JSON envelope

### M1: Parsers + IR

- [ ] `parser/org`: parse heading structure + properties drawer + scheduled/deadline + ID
- [ ] `parser/md`: parse headings + task list + frontmatter `id` (optional) + links
- [ ] sha256 calculation for documents
- [ ] return `ir.Document` from parsers

Definition of done:
- [ ] `orgx file parse <path> --json` returns `ir.Document`
- [ ] tests with fixture files (Org + Markdown)

### M2: Outline + refs

- [ ] `orgx file outline`: fast heading scan, `--max-depth`
- [ ] `ParseRefFromArg`:
  - [ ] split `path::selector` by last `::`
  - [ ] support `ID:`, `/Outline`, `H:`
  - [ ] support `@ws/` prefix
- [ ] ref normalization in outputs

Definition of done:
- [ ] `orgx heading view <ref>` resolves ID refs reliably
- [ ] errors are typed and stable (`E_NOT_FOUND_REF`, etc)

### M3: Workspace commands

- [ ] `orgx init`: create config dir + starter config
- [ ] `orgx ws add/list/show/use/rm`
- [ ] config write is atomic

Definition of done:
- [ ] `orgx ws add work --root /tmp/x` then `orgx ws use work` works
- [ ] tests for config round-trip

### M4: Heading list/view

- [ ] `orgx heading list [<path>]` with filters: `--level --todo --tag --limit`
- [ ] output:
  - [ ] human: table/tree (minimal)
  - [ ] json: list of headings with stable refs

Definition of done:
- [ ] deterministic ordering and stable json fields

### M5: First write command (`heading set`)

- [ ] compute patch (do not regenerate whole file)
- [ ] `--dry-run` prints diff (stderr) and returns `changes[]`
- [ ] `--yes` applies with backup + atomic write
- [ ] interactive confirm only when `IO.CanPrompt()`

Definition of done:
- [ ] modifies TODO/tags/props without destroying unrelated formatting
- [ ] tests: dry-run output + applied output; backup created

### M6+: Move/append/search/convert/roam/agenda/capture

Gate each feature behind:
- [ ] spec section
- [ ] fixtures + unit tests
- [ ] error codes + JSON fields

---

## Per-command checklist (use for every new command)

- [ ] `pkg/cmd/<domain>/<cmd>/` package with `Options` struct
- [ ] `NewCmdX(f, runF)` with injected deps set from Factory
- [ ] validate flags/args early; return `cmdutil.FlagError` for usage-worthy issues
- [ ] no global state; no direct `os.Stdout`/`os.Stderr` (use IOStreams)
- [ ] JSON mode:
  - [ ] stdout is JSON only
  - [ ] stable envelope, stable ordering
- [ ] human mode:
  - [ ] minimal output; stderr for hints
- [ ] tests:
  - [ ] table-driven parsing tests
  - [ ] command execution tests (`cmd.ExecuteC`) using `iostreams.Test()`
- [ ] docs:
  - [ ] update `docs/SPEC.md` command section if behavior changed

---

## Error code checklist

For each failure path, pick one code and stick to it:
- [ ] `E_NO_WORKSPACE`
- [ ] `E_NOT_FOUND_REF`
- [ ] `E_PARSE_FAILED`
- [ ] `E_WRITE_FORBIDDEN`
- [ ] `E_CONFLICT_LOCKED` (future)
- [ ] `E_INVALID_INPUT` (generic)

---

## Determinism checklist

- [ ] sort results and errors before printing
- [ ] avoid time in outputs unless explicitly requested
- [ ] avoid absolute paths unless needed (prefer `@ws/relpath` when possible)

