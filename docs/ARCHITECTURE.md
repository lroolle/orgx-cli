# orgx Architecture

The frame, top to bottom. Each layer only knows the one below it.

```
contract    what agents and scripts rely on
            --json envelopes (kind: orgx.<command>.v1, docs/schema.md),
            orgx.error.v1 with runnable fix, exit codes, embedded
            SKILL.md (orgx skills install) + sync-guard test

vault       the knowledge graph on disk (pkg/roam)
            layout, discovery, nodes, journals, derived graph
            commands: init, node, daily, graph (+ find/backlinks
            defaulting into the vault)

documents   precise operations on one file or ref (pkg/cmd/*)
            peek, get, set, find, capture, promote, log, ls, links,
            id; refs (::ID: stable, ::/outline and ::H: fragile);
            LOGBOOK state logging, org timestamps (pkg/orgtime)

substrate   parsing and text (pkg/parser, pkg/ir, pkg/textutil)
            go-org + goldmark -> one IR; line tracking for surgical
            writes; CRLF preservation
```

Command plumbing is gh-style throughout: `Factory` -> `Options` ->
`runF` injection, `iostreams` for testable I/O, one `Exporter` for
JSON (`pkg/cmdutil`).

## The vault

A vault is a directory marked by `.orgx/` at its root. Discovery
walks up from the current directory the way git finds a repository,
so inside the tree no command needs `--root`, `-w`, or `--in`.
Resolution order everywhere: `--root` flag > named workspace (`-w`,
typos error rather than fall through) > discovered vault > default
workspace > a fix-bearing error.

The default layout is Logseq's, because it earned it:

```
vault/
  .orgx/           marker; config.yaml (layout overrides only)
  journals/        time-indexed: one node per day, append-only
  pages/           topic-indexed: one node per subject
  whiteboards/     reserved for spatial notes (no tooling yet)
  assets/          images and attachments
  pages/contents.org     the front door
  pages/flashcards.org   the always-loaded page: durable facts,
                         preferences, invariants
```

Two deliberate absences:

- **No `graph/` directory.** The graph (nodes, edges, broken links)
  is derived from the files by `orgx graph` on demand. Derived data
  stored in a vault goes stale and then lies; org-roam needs a
  sqlite cache because Emacs must be interactive — a CLI does not.
- **No database at all.** Files are the database. Everything orgx
  knows is re-derivable by reading the directory, which is also why
  an existing org-roam vault works unmodified (file-level `:ID:` +
  `#+title` is the shared shape; its `daily/` convention is
  detected and respected).

## Nodes and the graph

A node is an org file whose head carries `:ID:` (drawer) and
`#+title`. `pkg/roam` reads file heads only (`ReadMeta`/`Scan`) —
listing a thousand-node vault costs milliseconds, not a parser pass
per file. Full parsing happens only where content is needed
(get/graph/backlinks).

Links are `[[id:<uuid>]]`, in body text or headline titles (journal
entries ARE headings, so title links backlink — the parser extracts
both). `orgx graph` resolves every id link against the node set:
matches become edges, misses are reported as broken. Nothing is
cached.

## Agents living in the graph

Agents are residents, not tools bolted on:

- They journal completed work with `orgx daily "..." --as <name>`;
  the author rides as an `@name` tag on the entry heading, so
  `orgx find --tag @claude` is the agent's audit trail — in the
  same file the human reads in Emacs.
- `pages/flashcards.org` is the standing context: agents read it
  first, and write to it sparingly (durable facts, not chatter).
- Consent is structural: reads never write; every write requires
  `--yes` off-TTY and lands in a plain text file the human can
  diff, edit, or revert.

Next steps in this direction (roadmap): heading-level nodes, agent
home pages, and an embedded agent (agentkit) once the substrate
earns it.

## Invariants

1. READS NEVER WRITE.
2. Writes are explicit (`--yes` off-TTY) and CRLF-preserving.
3. `--json` stdout is valid JSON only; diagnostics go to stderr.
4. Envelope kinds are versioned; changes within v1 are additive.
5. Output is deterministic: sorted, stable schema, no randomness.
6. The vault is Emacs-compatible plain text; orgx never writes a
   format only orgx can read.
