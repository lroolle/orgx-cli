# orgx-cli

```
  ___  _ __ __ ___  __
 / _ \| '__/ _` \ \/ /
| (_) | | | (_| |>  <
 \___/|_|  \__, /_/\_\
           |___/
```

**Your org-mode vault, for humans and AI agents.**
Journals, pages, links, backlinks, a derived graph — plain org
files Emacs reads natively, with a CLI contract agents can drive.

## Why

Knowledge tools split the world: Emacs/org-roam is for you, agent
tooling is for the model, and nothing is shared. orgx makes one
graph both can live in:

- **You** keep org-mode: stable `:ID:`s, TODO states, LOGBOOK
  history, org-roam-compatible files. Nothing orgx writes is a
  format only orgx can read.
- **Agents** get a contract: token-efficient structure views,
  versioned JSON envelopes, errors that carry their own fix, and a
  skill that installs from the binary.
- **Together**: an agent journals its work with `orgx daily --as
  claude` into the same file you read at the end of the day, links
  the nodes it touched, and `orgx find --tag @claude` is its audit
  trail.

## Install

```bash
go install github.com/lroolle/orgx-cli/cmd/orgx@latest
```

Or grab a binary from [Releases](https://github.com/lroolle/orgx-cli/releases),
or build from source: `git clone ... && make build && make install`.

## The vault

```bash
orgx init ~/org/vault && cd ~/org/vault
```

```
vault/
  .orgx/           marker — inside the tree, every command finds
                   the vault by walking up, like git; no flags
  journals/        one org file per day        (orgx daily)
  pages/           one node per topic          (orgx node new)
  whiteboards/     reserved for spatial notes
  assets/          images, attachments
  pages/contents.org     the front door
  pages/flashcards.org   durable facts — agents read this first
```

The layout is Logseq's, the file format is org-roam's — an
existing org-roam directory is already a valid vault (`orgx init`
inside one only adds what's missing, and its `daily/` convention
is respected). The graph is derived, never cached: files are the
database.

```bash
orgx node new "SRP protocol notes" --tags auth --yes
orgx daily "reserved 3 aliases for signup flows" --as claude --yes
orgx daily "read [[id:<node-id>][SRP notes]], looks solid" --yes

orgx node list --json          # every node, files-without-ID counted
orgx backlinks <node-id>       # who references this?
orgx graph --json              # nodes, edges, and BROKEN links
orgx find "" --tag @claude     # everything the agent did, dated
```

## Documents, precisely

The original orgx core: work on structured text without paying for
whole files.

```bash
orgx peek notes.org                     # outline only, ~100 tokens
orgx get "notes.org::ID:abc123"         # one section, not 1000 lines
orgx set "notes.org::ID:abc123" --todo DONE --yes   # surgical, logged
orgx find "design" --todo TODO,STRT     # refs, not content
```

GTD flows through the same refs — `capture` into an inbox,
`promote` between files, LOGBOOK state history via `orgx log`,
org timestamps with relative dates (`--deadline +3d`).

## Agent contract

- Every command takes `--json`; output is a versioned envelope
  (`kind: orgx.<command>.v1`, lists as `{kind, count, items}`) —
  registry in [docs/schema.md](docs/schema.md), changes are
  additive within v1.
- Errors under `--json` are `orgx.error.v1` on stderr with a
  runnable `error.fix`; stdout stays data-only. Reads never write;
  writes require `--yes` off-TTY.
- `orgx skills install` (`--scope user|project`) writes the
  embedded [SKILL.md](skills/orgx/SKILL.md) where agent runtimes
  look — skill text and command surface ship together, enforced by
  a sync-guard test.

## Docs

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — the four-layer
  frame: contract / vault / documents / substrate, and why the
  graph has no database
- [docs/schema.md](docs/schema.md) — JSON kind registry
- [skills/orgx/SKILL.md](skills/orgx/SKILL.md) — the agent manual
- [DEVLOG.org](DEVLOG.org) — design principles, session history

## Acknowledgments

- [gh-cli](https://github.com/cli/cli) — command architecture
- [org-roam](https://www.orgroam.com) — the node shape
- [Logseq](https://logseq.com) — the vault layout
- [go-org](https://github.com/niklasfasching/go-org), [goldmark](https://github.com/yuin/goldmark) — parsers

## License

[MIT](LICENSE)
