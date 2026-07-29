# JSON Schema Contract

Every `--json` output is an envelope carrying a versioned `kind`,
derived from the command path (`orgx node list` →
`orgx.node.list.v1`). Object outputs carry `kind` inline; list
outputs are wrapped:

```json
{ "kind": "orgx.find.v1", "count": 2, "items": [ ... ] }
```

Errors under `--json` go to **stderr** as `orgx.error.v1`; stdout
stays data-only:

```json
{ "kind": "orgx.error.v1",
  "error": { "message": "...", "fix": "<runnable command>" } }
```

## Policy

- Within a `v1` kind, changes are **additive only** — new fields
  may appear, existing fields keep their meaning and type.
- A breaking change bumps the kind's version (`.v2`); the old kind
  keeps working for at least one release.
- Empty results are `count: 0` with an empty `items` array, never
  `null`.

## Kinds

| Kind | Payload |
|------|---------|
| `orgx.peek.v1` | file outline: path, lines, headings[] |
| `orgx.get.v1` | one section by ref |
| `orgx.find.v1` | heading matches: ref, title, todo, tags, dates |
| `orgx.set.v1` | edit result |
| `orgx.capture.v1` | ref, path, title, todo, id |
| `orgx.promote.v1` | moved ref, state transition |
| `orgx.log.v1` | state history entries |
| `orgx.ls.v1` | files with stats |
| `orgx.links.v1` | outgoing links |
| `orgx.backlinks.v1` | incoming links: source, target, title |
| `orgx.init.v1` | root, created[], kept[] |
| `orgx.node.new.v1` | path, id, title |
| `orgx.node.list.v1` | root, count, skipped, nodes[] |
| `orgx.daily.v1` | path, date, created, entry — or day content |
| `orgx.graph.v1` | root, nodes[], edges[{from,to}], broken[] |
| `orgx.skills.list.v1` | embedded skills |
| `orgx.skills.install.v1` | path, scope, updated |
| `orgx.error.v1` | error.message, error.fix (stderr) |

Sub-command kinds under `file`, `heading`, `id`, and `ws` follow
the same derivation (`orgx.ws.list.v1`, ...).
