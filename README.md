# orgx-cli

```
  ___  _ __ __ ___  __
 / _ \| '__/ _` \ \/ /
| (_) | | | (_| |>  <
 \___/|_|  \__, /_/\_\
           |___/
```

**Org-mode and Markdown CLI for humans and AI agents — and our own
roam built on it.**

## Why

AI agents work well with Markdown but your knowledge base runs on
Org-mode. You want stable IDs, TODO states, scheduled dates,
org-roam links. Agents want deterministic outputs, structured JSON,
safe edits.

orgx bridges both, and the roam layer makes the workspace a shared
knowledge graph that agents live in: they journal their work with
`orgx daily --as <name>`, link nodes by ID, and everything they do
is a plain org file the human reads in Emacs.

## Install

```bash
go install github.com/lroolle/orgx-cli/cmd/orgx@latest
```

Or:

```bash
git clone https://github.com/lroolle/orgx-cli.git
cd orgx-cli && make build && make install
```

## Usage

```bash
# Documents
orgx peek notes.org                 # structure only, ~100 tokens
orgx get "notes.org::ID:8f3c..."    # one section, not the file
orgx set "notes.org::ID:8f3c..." --todo DONE --yes

# GTD
orgx capture "Review API docs" --yes
orgx promote INBOX.org::ID:abc --to BACKLOG.org --todo TODO --yes
orgx log BACKLOG.org::ID:abc

# Roam (an existing org-roam directory already works)
orgx ws add main --root ~/org/roam && orgx ws use main
orgx node new "SRP protocol notes" --tags auth --yes
orgx daily "reserved 3 aliases for signup flows" --as claude --yes
orgx find "" --tag @claude          # what did the agent do?
orgx backlinks <node-id>            # who references this?
```

## Agent contract

- Every command takes `--json`; output is a versioned envelope
  (`kind: orgx.<command>.v1`, lists as `{kind, count, items}`).
- Errors under `--json` are `orgx.error.v1` on stderr with a
  runnable `fix` when one exists; stdout stays data-only.
- Reads never write; writes require `--yes` off-TTY.
- `orgx skills install` writes the embedded SKILL.md where agent
  runtimes look — skill text and command surface ship together, and
  a sync-guard test keeps them honest.

## Docs

- [CLAUDE.md](CLAUDE.md) - architecture
- [docs/README.md](docs/README.md) - specs and plan
- [skills/orgx/SKILL.md](skills/orgx/SKILL.md) - command reference

## Acknowledgments

- [gh-cli](https://github.com/cli/cli)
- [atlas-cli](https://github.com/lroolle/atlas-cli)
- [go-org](https://github.com/niklasfasching/go-org)
- [goldmark](https://github.com/yuin/goldmark)
