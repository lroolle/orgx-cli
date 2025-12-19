# orgx CLI

Workspace-aware CLI for Org-mode and Markdown. For humans and AI agents.
Org as canonical store, Markdown as interface. Shared IR, deterministic outputs.

---

## Why AI Agents Love gh-cli (And We Copy It)

AI agents use `gh` extremely well. Why:

1. **Flexible arg parsing** - accepts ID, URL, or ref
   ```go
   // gh pattern: ParseIssueFromArg handles "123", "#123", "https://..."
   // orgx pattern: ParseRefFromArg handles "path::ID:uuid", "path::/Outline", URL
   ```

2. **Predictable output** - same command, same structure
   ```bash
   gh issue view 123 --json title,state    # always JSON
   orgx heading view ref --json            # always JSON
   ```

3. **Non-interactive by default** - agents can't type "y"
   ```go
   if opts.IO.CanPrompt() && !opts.Confirmed {
       // only prompt humans at TTY
   }
   ```

4. **`--yes` flag** - skip confirmation for automation
   ```bash
   gh issue delete 123 --yes
   orgx heading set ref --todo DONE --yes
   ```

5. **Consistent verbs** - list, view, create, edit, delete
   ```
   gh issue list/view/create/edit/delete
   orgx heading list/view/set/move/append
   ```

---

## Architecture Overview

```
cmd/orgx/main.go              # Entry point
pkg/
├── cmd/
│   ├── factory/              # Factory construction
│   ├── root/                 # Root command
│   ├── file/                 # file parse/outline/search
│   │   ├── file.go           # Aggregator
│   │   ├── parse/            # Subcommand
│   │   ├── outline/
│   │   └── shared/           # Domain-specific shared utils
│   ├── heading/
│   │   ├── heading.go
│   │   ├── list/
│   │   ├── view/
│   │   ├── set/
│   │   └── shared/           # ParseRefFromArg lives here
│   ├── ws/
│   └── [other domains]/
├── cmdutil/                  # Command utilities
│   ├── factory.go            # Factory interface
│   ├── errors.go             # FlagError, SilentError
│   ├── json_flags.go         # Exporter interface
│   └── cmdgroup.go           # AddGroup helper
├── iostreams/                # Testable I/O
├── ir/                       # Intermediate Representation
├── parser/
│   ├── org/                  # go-org wrapper
│   └── md/                   # goldmark wrapper
└── config/                   # Workspace config
internal/
└── workspace/                # Workspace resolution logic
```

---

## Factory Pattern

### Factory Interface

```go
// pkg/cmdutil/factory.go
type Factory struct {
    AppVersion string
    IOStreams  *iostreams.IOStreams
    Config     func() (Config, error)      // Lazy
    Workspace  func() (*Workspace, error)  // Lazy, depends on Config
    Prompter   Prompter
}
```

### Factory Construction

```go
// pkg/cmd/factory/default.go
func New(appVersion string) *cmdutil.Factory {
    f := &cmdutil.Factory{
        AppVersion: appVersion,
    }

    // Dependency order matters
    f.Config = configFunc()                  // No deps
    f.IOStreams = ioStreams(f)               // Depends on Config
    f.Workspace = workspaceFunc(f)           // Depends on Config
    f.Prompter = newPrompter(f)              // Depends on IOStreams

    return f
}
```

Key: Functions are lazy-initialized in dependency order.

---

## IOStreams Abstraction

```go
// pkg/iostreams/iostreams.go
type IOStreams struct {
    In     io.Reader
    Out    io.Writer
    ErrOut io.Writer

    colorEnabled    bool
    stdinTTY        bool
    stdoutTTY       bool
    stderrTTY       bool
    neverPrompt     bool
    pagerCommand    string
}

func (s *IOStreams) ColorEnabled() bool
func (s *IOStreams) IsStdoutTTY() bool
func (s *IOStreams) CanPrompt() bool  // stdoutTTY && !neverPrompt

// Test helpers
func Test() (ios *IOStreams, stdin *bytes.Buffer, stdout *bytes.Buffer, stderr *bytes.Buffer)
func (s *IOStreams) SetStdoutTTY(v bool)
func (s *IOStreams) SetColorEnabled(v bool)
```

Not an interface - concrete struct with test mode.

---

## Options Struct Pattern

Every subcommand defines its own Options struct:

```go
// pkg/cmd/heading/set/set.go
type SetOptions struct {
    // Injected dependencies
    IO        *iostreams.IOStreams
    Config    func() (config.Config, error)
    Workspace func() (*workspace.Workspace, error)
    Prompter  cmdutil.Prompter

    // Flags
    Ref       string
    Todo      string
    Tags      []string
    Props     map[string]string
    DryRun    bool
    Confirmed bool

    // JSON output
    Exporter  cmdutil.Exporter
}
```

Split into:
1. **Injected deps** - from Factory
2. **Flag fields** - CLI flags
3. **Exporter** - for --json

---

## Command Constructor with runF Injection

```go
// pkg/cmd/heading/set/set.go
func NewCmdSet(f *cmdutil.Factory, runF func(*SetOptions) error) *cobra.Command {
    opts := &SetOptions{
        IO:        f.IOStreams,
        Config:    f.Config,
        Workspace: f.Workspace,
        Prompter:  f.Prompter,
    }

    cmd := &cobra.Command{
        Use:   "set <ref>",
        Short: "Set heading properties",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            opts.Ref = args[0]

            if runF != nil {
                return runF(opts)  // Test injection
            }
            return setRun(opts)
        },
    }

    cmd.Flags().StringVar(&opts.Todo, "todo", "", "Set TODO state")
    cmd.Flags().StringSliceVar(&opts.Tags, "tags", nil, "Set tags (+add, -remove)")
    cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes")
    cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Skip confirmation")

    cmdutil.AddJSONFlags(cmd, &opts.Exporter, headingFields)

    return cmd
}
```

### Test Usage

```go
func TestSet(t *testing.T) {
    ios, _, stdout, _ := iostreams.Test()
    ios.SetStdoutTTY(false)

    f := &cmdutil.Factory{
        IOStreams: ios,
        Config: func() (config.Config, error) {
            return config.NewBlank(), nil
        },
    }

    cmd := set.NewCmdSet(f, func(opts *set.SetOptions) error {
        // Override before run
        opts.DryRun = true
        return set.SetRun(opts)
    })

    cmd.SetArgs([]string{"notes.org::ID:abc", "--todo", "DONE"})
    _, err := cmd.ExecuteC()
    assert.NoError(t, err)
}
```

---

## Command Aggregator Pattern

```go
// pkg/cmd/heading/heading.go
func NewCmdHeading(f *cmdutil.Factory) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "heading <command>",
        Short: "Manage headings",
    }

    cmdutil.AddGroup(cmd, "Query",
        list.NewCmdList(f, nil),
        view.NewCmdView(f, nil),
    )

    cmdutil.AddGroup(cmd, "Modify",
        set.NewCmdSet(f, nil),
        move.NewCmdMove(f, nil),
        append.NewCmdAppend(f, nil),
    )

    return cmd
}
```

### AddGroup Helper

```go
// pkg/cmdutil/cmdgroup.go
func AddGroup(parent *cobra.Command, title string, cmds ...*cobra.Command) {
    g := &cobra.Group{Title: title, ID: title}
    parent.AddGroup(g)
    for _, c := range cmds {
        c.GroupID = g.ID
        parent.AddCommand(c)
    }
}
```

---

## Shared Utilities (Per Domain)

```go
// pkg/cmd/heading/shared/lookup.go
type Ref struct {
    Path    string
    RefType RefType  // ID, Outline, Hash
    Value   string
}

func ParseRefFromArg(arg string) (Ref, error) {
    // Try URL first
    if ref, ok := tryParseURL(arg); ok {
        return ref, nil
    }
    // Try path::ID:uuid
    if ref, ok := tryParseIDRef(arg); ok {
        return ref, nil
    }
    // Try path::/Outline/Path
    if ref, ok := tryParseOutlineRef(arg); ok {
        return ref, nil
    }
    // Try path::H:hash
    if ref, ok := tryParseHashRef(arg); ok {
        return ref, nil
    }
    // Fallback: treat as file path
    return Ref{Path: arg}, nil
}
```

Shared code lives in domain, not generic utils.

---

## JSON Output (Exporter)

```go
// pkg/cmdutil/json_flags.go
type Exporter interface {
    Write(io *iostreams.IOStreams, data interface{}) error
    Fields() []string
}

func AddJSONFlags(cmd *cobra.Command, target *Exporter, defaultFields []string)
```

### Field Definitions

```go
// pkg/cmd/heading/shared/fields.go
var HeadingFields = []string{
    "ref",
    "level",
    "title",
    "todo",
    "tags",
    "props",
    "scheduled",
    "deadline",
    "body",
    "span",
}
```

### Usage in Command

```go
func listRun(opts *ListOptions) error {
    headings, err := fetchHeadings(opts)
    if err != nil {
        return err
    }

    if opts.Exporter != nil {
        return opts.Exporter.Write(opts.IO, headings)
    }

    // Human-readable output
    return renderHeadings(opts.IO, headings)
}
```

---

## Error Handling

```go
// pkg/cmdutil/errors.go

// FlagError - triggers usage display
type FlagError struct {
    err error
}

func FlagErrorf(format string, args ...interface{}) error
func FlagErrorWrap(err error) error

// Signals
var SilentError = errors.New("SilentError")  // Exit 1, no message
var CancelError = errors.New("CancelError")  // User cancelled
```

### Usage

```go
if opts.Ref == "" {
    return cmdutil.FlagErrorf("ref argument required")
}

// Main handles FlagError specially
if errors.As(err, &cmdutil.FlagError{}) {
    cmd.Usage()
}
```

---

## Confirmation Pattern

```go
func setRun(opts *SetOptions) error {
    // 1. Parse ref
    ref, err := shared.ParseRefFromArg(opts.Ref)
    if err != nil {
        return err
    }

    // 2. Compute changes
    changes, err := computeChanges(ref, opts)
    if err != nil {
        return err
    }

    // 3. Dry run: show diff and exit
    if opts.DryRun {
        return printDiff(opts.IO, changes)
    }

    // 4. Interactive confirmation (TTY only)
    if opts.IO.CanPrompt() && !opts.Confirmed {
        fmt.Fprintf(opts.IO.ErrOut, "Will modify: %s\n", opts.Ref)
        if err := opts.Prompter.Confirm("Apply changes?"); err != nil {
            return cmdutil.CancelError
        }
    }

    // 5. Apply
    return applyChanges(changes)
}
```

---

## Parsers

**Org**: [niklasfasching/go-org](https://github.com/niklasfasching/go-org)
- 80/20 Org subset, Hugo uses it, good enough

**Markdown**: [yuin/goldmark](https://github.com/yuin/goldmark)
- CommonMark, extensible AST, frontmatter via plugin

Don't roll your own parser. You will regret it.

---

## IR (Intermediate Representation)

Both `.org` and `.md` parse to this. Agents work with IR, not raw syntax.

```
Document
├── path, sha256, doc_type: "org"|"md"
├── meta: { title?, frontmatter: {k:v} }
└── nodes: Node[]

Heading
├── ref: "path::ID:uuid" | "path::H:hash" | "path::/Outline/Path"
├── level, title, todo, tags[], props{}, scheduled, deadline
├── body: { raw, blocks[] }
├── children: Node[]
└── span: { start, end }

Task: ref, state: "open"|"done", text, tags, dates, span
Link: kind: "http"|"file"|"id"|"roam", target, desc
Block: kind: "code"|"quote"|"table"|"other", raw, span
```

### Stable References

Agents use refs, not line numbers.

| Format | Pattern | Stability |
|--------|---------|-----------|
| Org | `path::ID:<uuid>` | Stable |
| Org | `path::/Outline/Path` | Fragile |
| Md | `path::H:<hash>` | Fragile |
| Md | frontmatter `id: uuid` | Stable if maintained |

---

## Commands

```
orgx
├── init
├── ws add/list/show/use
├── file parse/outline/search/stats
├── heading list/view/set/move/append
├── roam node search/view, backlinks, node create
├── capture add, templates
├── agenda list/next/overdue
├── convert md2org/org2md
└── export md/html
```

### Global Flags
```
--ws <name>      workspace
--root <path>    override root
--json           machine output
--format org|md  output format
--dry-run        preview changes
--yes            non-interactive
```

---

## Output Contract

Human output: short, terminal-friendly.
JSON output (`--json`): stable schema, agents parse this.

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

---

## Editing

Patch-based. No surprise mutations.

1. Parse → IR
2. Transform → IR'
3. Generate diff
4. `--dry-run`: show diff
5. `--yes`: apply with backup

Backup: `file.org~20251219T152300`

---

## Workspace Config

```yaml
# ~/.config/orgx/config.yaml
version: 1
default_workspace: work
workspaces:
  work:
    root: /Users/eric/org
    roam_db: /Users/eric/.emacs.d/org-roam.db
    inbox: /Users/eric/org/inbox.org
  notes:
    root: /Users/eric/notes
    format: md
```

---

## Error Codes

```
E_NO_WORKSPACE      no workspace configured
E_NOT_FOUND_REF     ref not found
E_PARSE_FAILED      parse error
E_CONFLICT_LOCKED   file locked
E_WRITE_FORBIDDEN   write not allowed
```

---

## Dependencies

```go
require (
    github.com/niklasfasching/go-org v1.7.0
    github.com/yuin/goldmark v1.7.8
    go.abhg.dev/goldmark/frontmatter v0.2.0
    github.com/spf13/cobra v1.8.1
    github.com/spf13/viper v1.19.0
    modernc.org/sqlite v1.28.0  // org-roam db
)
```

---

## Implementation Order

Start with read-only commands, add writes later.

1. **iostreams** - testable I/O first
2. **cmdutil** - Factory, errors, AddGroup
3. **file parse** - first real command
4. **file outline** - tree view
5. **heading list** - filtering
6. **heading view** - ref lookup
7. **ws** - workspace management
8. **heading set** - first write command
9. Continue per need

---

## References

- [gh-cli](https://github.com/cli/cli) - CLI patterns
- [atlas-cli](https://github.com/lroolle/atlas-cli) - AI-first CLI, skill bundle
- [go-org](https://github.com/niklasfasching/go-org) - Org parser
- [goldmark](https://github.com/yuin/goldmark) - Md parser
- [logseq/mldoc](https://github.com/logseq/mldoc) - dual parser reference
