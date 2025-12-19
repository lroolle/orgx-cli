# orgx-cli

```
  ___  _ __ __ ___  __
 / _ \| '__/ _` \ \/ /
| (_) | | | (_| |>  <
 \___/|_|  \__, /_/\_\
           |___/
```

**Org-mode and Markdown CLI for humans and AI agents.**

## Why

AI agents work well with Markdown but your knowledge base runs on Org-mode.
You want stable IDs, TODO states, scheduled dates, org-roam links.
Agents want deterministic outputs, structured JSON, safe edits.

orgx bridges both.

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
orgx file parse notes.org --json
orgx heading list notes.org --level 2
orgx heading view notes.org::ID:8f3c...

orgx ws add work --root ~/org
orgx convert md2org draft.md --to inbox.org
```

## Docs

- [CLAUDE.md](CLAUDE.md) - architecture
- [docs/README.md](docs/README.md) - specs and plan
- [skills/orgx/SKILL.md](skills/orgx/SKILL.md) - command reference

## Acknowledgments

- [gh-cli](https://github.com/cli/cli)
- [atlas-cli](https://github.com/lroolle/atlas-cli)
- [go-org](https://github.com/niklasfasching/go-org)
- [goldmark](https://github.com/yuin/goldmark)
