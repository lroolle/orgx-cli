# The Agent Frame

orgx in a world where the agents already have hands.

Written 2026-07-29 against orgx v0.2.0 and its sibling ihme-cli
v0.5.0 (the L3 reference: embedded agent, agentkit kernel, consent
gates, file memory). This doc records the comparison, the frame we
commit to, and five decisions with their traps. Companion to
ARCHITECTURE.md (the layer stack); the maturity-ladder vocabulary
(L0 machine-readable → L3 agent-embedded) comes from the agentic-CLI
practices distilled out of ihme-cli.

## Where the two CLIs stand

|                      | ihme-cli                          | orgx                          |
|----------------------|-----------------------------------|-------------------------------|
| Domain               | Apple HME API: remote, authed     | Plain files: local, diffable  |
| Maturity ladder      | L3 agent-embedded                 | L1 agent-operable             |
| Agent code share     | 64% of 13.2k lines                | 0% (no LLM code at all)       |
| Memory               | Own mini-vault (internal/memory)  | The vault IS the product      |
| Consent model        | Gate in code + consent cards      | Reads-never-write + `--yes`;  |
|                      |                                   | git is the review surface     |
| Moat for an agent    | Private API + taste rubric        | None yet: every head already  |
|                      |                                   | has file hands                |

The domains explain the ladder gap. ihme needed an embedded agent
because its work lives behind an authenticated API with judgment
calls (address taste) and semi-irreversible cloud state (consent).
No generic agent could do that job with Read/Edit. orgx's work is
files — the one substrate every coding agent already manipulates
natively. The moat that justified ihme's 8.5k agent lines does not
exist here, and pretending it does would buy us a second head with
nothing only it can do.

The pivotal observation: ihme's memory directory
(`journals/YYYY_MM_DD.md`, `pages/<slug>.md`, `pages/flashcards.md`)
is a proto-orgx-vault. ihme built a small one inside itself because
it needed one; orgx is that idea grown into the product. The two
projects converge on the same shape from opposite ends — which tells
us the shape is right, and tells us what orgx is *for*: it is the
substrate agents remember and reason on, not another reasoner.

## The frame

```
heads       Claude Code, codex, humans-in-Emacs, (someday: orgx agent)
            think, judge, and edit freely with their own editors

contract    SKILL.md + JSON envelopes + fix-bearing errors + refs
            the interface any head drives orgx through

hands       invariant-preserving writes: daily, capture, node new,
            set --todo/--tags, promote — shape correctness (IDs,
            LOGBOOK, timestamps, CRLF) that freehand edits get wrong

eyes        derived truth: peek, find, graph, backlinks, id check —
            whole-vault facts no head computes by hand, at token
            prices no Read call matches

vault       files on disk — THE memory; no second memory system.
            readable by Emacs, Logseq, and (decision 4) orgx serve
```

One sentence: **orgx supplies hands and eyes to external heads; it
does not grow its own head until a trigger fires** (decision 3).
This is ihme practice #5 inverted — ihme made the agent "just
another adapter over the service"; orgx makes every agent an adapter
over the vault.

## Decision 1 — What is agent-native today, and what leaks

Native and earned (keep compounding):

- Versioned envelopes (`orgx.<cmd>.v1`), bare arrays wrapped, never
  `null` items; `orgx.error.v1` with a runnable fix, on stderr.
- Reads never write; every write takes `--yes` off-TTY; dry-run
  diffs on stderr, data on stdout.
- Embedded SKILL.md + `skills install` + the sync-guard test.
- Token-shaped reads: peek/find return refs not content, `# ->`
  continuation hints, head-only metadata scan (ReadMeta).
- Residency: `--as` author tags, `@name` audit trail via find,
  `pages/flashcards.org` as standing context.

Leaks (each concrete, each cheap):

1. **`set` and `heading set` have no `--json`.** The write path is
   exactly where an agent needs machine confirmation of what
   changed. schema.md already registers `orgx.set.v1` — the flag
   was never wired. This is the single worst gap.
2. `ws add`, `ws use`, `version` have no `--json`.
3. No `ORGX_ROOT` / `ORGX_WS` env overrides (planned in PLAN.md,
   unbuilt). Agents running in subshells and hooks want env, not
   flags threaded through every call.
4. No typed error kinds. `error.v1` carries a fix string but no
   stable code an agent can branch on (PLAN.md's `E_*` taxonomy was
   never built). Additive within v1: add a `code` field.
5. Sync-guard checks top-level command names only; subcommands and
   flags can drift out of SKILL.md silently.
6. Dead config: `RoamDB`, `Inbox`, `Format` are stored and shown
   but never read. Wire `Inbox` into capture or cut all three —
   fields that lie are worse than fields that are missing.
7. `pkg/roam` scans `.org` only, so a markdown Logseq graph (e.g.
   ihme's own memory dir) has files but no nodes. Decide: md nodes
   in Scan, or document the boundary.
8. PLAN.md is stale (shipped items unchecked, references a
   `docs/SPEC.md` that does not exist). Stale plans read as truth
   to agents; retire it into DEVLOG.org's roadmap.

## Decision 2 — Multi-graph: the IDs are already global

Framing: how do two vaults link, and how does one invocation see
more than one graph?

Spread considered:

1. *One-graph doctrine.* Do nothing; Logseq's own answer. Cost:
   work/personal/agent-memory graphs can never reference each other.
2. *Union resolution.* `[[id:<uuid>]]` already names no vault —
   UUIDs are globally unique, and Emacs org-id resolves them across
   all known directories via org-id-locations. Make orgx do the
   same: a link that misses in the current vault is resolved
   against the other registered workspaces before being called
   broken.
3. *New syntax.* `@ws/` ref prefixes or `[[orgx:ws/id]]` links.
   Trap: violates invariant 6 — a link format only orgx can read;
   Emacs and org-roam cannot follow it. Dead on arrival.
4. *Physical merge.* Symlink vaults under one root. Trap: mixes
   lifecycles and privacy (work vs personal vs agent scratch), and
   breaks each vault's standalone Emacs story.

Commit: **option 2 — cross-workspace ID resolution, no new
syntax.** The org ecosystem solved this decades ago: IDs are
global, resolvers are multi-root. Concretely:

- `orgx graph` classifies edges `internal` | `external` (with a
  `ws`/`root` field) | `broken` — a link into another registered
  workspace stops being reported as broken. Additive within
  `orgx.graph.v1`.
- `backlinks --all-workspaces` (and later `find`): scan the union
  of registered roots. Head-only scan keeps this milliseconds even
  across several thousand nodes; no cache, no index file, so the
  no-database doctrine holds. If a vault someday grows past that,
  the fix is an org-id-locations-style derived cache — derived,
  regenerable, and only then.
- Graph-to-graph in practice: register an agent's memory directory
  as a workspace and its notes become linkable nodes in the same
  universe as the human vault (needs leak #7 fixed for md graphs).

## Decision 3 — The embedded agent: not yet, and here is the trigger

The high-stakes one. Full spread.

**A. Never embed; orgx is a tool.**
What: double down on hands and eyes; heads stay external forever.
Upside: every line compounds into the contract; zero provider/TUI
maintenance (ihme's adapter+TUI alone outweighs all of pkg/roam
tenfold). Cost: no standalone story; headless recurring vault work
(cron gardening) depends on an external agent CLI being present.

**B. Embed agentkit now (ihme's playbook).**
What: import/copy the ~600-line kernel, write an orgx adapter:
tools over vault ops, consent gate, BYOK config, TUI.
Upside: proven kernel; the vault-as-memory fit is *better* than
ihme's (no second memory system needed — the agent's memory IS the
product); standalone BYOK value.
Cost: ihme's own receipts say the adapter is where the mass is —
tools.go + gate.go + run.go + tui.go ≈ 3k lines before tests. And
orgx lacks the moat: every candidate task (summarize, garden,
refile, answer-from-vault) is one a generic head with the skill
already does well. We would maintain a second head that can do
nothing only it can do.

**C. Agent-bridged (L2): shell out to installed claude/codex.**
What: `orgx agent "<task>"` execs the user's agent CLI with the
skill and vault context preloaded (ccx's deliberate posture).
Upside: ~100 lines; the user's existing models and subscription;
best-available harness for free. Cost: hard dependency on an
external binary; consent lives in the foreign agent's permission
model, not ours.

**D. Inversion: build the residency substrate first.**
What: no head at all — build what makes ANY head a good resident:
agent home pages (a node per agent accumulating long-term notes),
heading-level nodes, and an `orgx context` command that assembles
flashcards tail + recent journals + query-relevant nodes into one
budgeted block (the `<memory>` injection ihme does in code,
exposed as a command any head can call).
Upside: serves Claude Code today and a future embedded agent
identically; it IS the "when the substrate earns it" work already
on the roadmap. Cost: nothing "acts" on its own yet.

Traps checked: A breaks on headless/cron use — but C covers that
for one hundred lines. B's trap is fatal today: no moat, big
adapter mass, and the elegance of an 8k-line codebase drowns in a
3k-line second head. C's trap (foreign consent model) is already
true for every Claude Code session in the vault now — the binding
protections are the vault's own: plain text, git, reads-never-write.
D's trap (built context nobody calls) is small: `orgx context` is
immediately consumable by the skill itself.

Commit: **D now, C as the cheap follow-up, B only on trigger.**
The trigger for B, written down so it is checkable and not vibes:

- a recurring headless vault task exists that the C bridge
  demonstrably serves badly (consent, latency, or availability), or
- agentkit is extracted to its own module, making B adapter-only
  cost, AND a concrete orgx-specific judgment loop exists that a
  generic head with the skill measurably fumbles.

Until a trigger fires, embedding is scope creep by ihme's own
definition ("additions that precede their trigger"). Prerequisite
either way, and worth doing regardless: keep vault operations
adapter-neutral (the runF bodies stay thin over pkg/roam), because
"the agent arrived in one release" at ihme only because the service
layer already existed.

## Decision 4 — Human preview: `orgx serve`, read-only

Spread considered:

1. *Nothing.* Emacs and Logseq already open the vault. Trap: the
   humans we actually want reviewing agent activity (non-Emacs
   collaborators, phone-adjacent quick checks) have no door.
2. *`orgx serve`* — localhost, read-only, everything derived
   per-request.
3. *Static export.* `orgx export --html`. Trap: stale the moment an
   agent journals; regeneration discipline never survives.
4. *Full webapp with editing.* Trap: violates the reads-never-write
   posture, duplicates what editors and agents do better, and is a
   permanent maintenance sink in a codebase whose taste is
   smallness.

Commit: **option 2.** Scope fence, deliberately tight:

- One command, `orgx serve`, localhost-bound by default.
- Server-rendered HTML, no build step, no JS framework. go-org
  ships an org->HTML writer; goldmark ships md->HTML — the
  rendering is already in our dependency tree.
- Surfaces: journals timeline (newest first), page view with a
  backlinks panel, node index, broken-links report, and a graph
  page fed by the same JSON `orgx graph` emits (a small embedded
  static asset may draw it; it reads the JSON, nothing more).
- **Zero write endpoints. Ever.** The web view is the human's
  reading surface; editing belongs to Emacs, Logseq, and agents,
  and review-of-writes belongs to git. This keeps invariant 1
  intact in the one place it would be most tempting to bend.
- Nothing cached: derive per request, exactly like `orgx graph`.
  A vault that outgrows per-request derivation is the same future
  problem as decision 2's, with the same answer, later.

## Decision 5 — Division of labor with file-editing heads

Heads edit files natively and well. The litmus for every proposed
orgx feature:

> Would a competent agent with Read/Edit do this correctly and
> cheaply without orgx? If yes, stay out. If no — whole-vault scan,
> shape-critical write, or token-expensive read — it is ours.

orgx does well (invest):

- Orientation cheaper than Read: peek, find, ls, get-by-ref.
- Derived truth no head computes by hand: graph, backlinks,
  id check, broken links.
- Shape-critical writes: daily, capture, node new, promote,
  set --todo/--tags — IDs, LOGBOOK, org timestamps, CRLF, the
  things freehand edits silently corrupt.
- Verification after freehand edits: the agent edits with its own
  editor, then `orgx id check` + `orgx graph` lint the result.
  "Agent edits, orgx verifies" is a first-class workflow, not a
  fallback — say so in the skill.
- The contract itself: stable refs, envelopes, fix-bearing errors.

orgx must not do:

- Arbitrary body editing (no `set --body`). The head's editor is
  better at prose surgery than any flag surface will ever be.
- A query language beyond find. Grep-shaped questions belong to
  the head's own tools; structured questions get structured
  commands.
- Sync, merge, conflict resolution. Git's job.
- Being the only door. Freehand edits stay first-class; orgx
  verifies after, never gates before. The moment orgx demands to
  mediate every write, it is a database with extra steps.
- A second memory system. The vault is the memory (see frame).

## Roadmap deltas

Feeding DEVLOG.org, in order:

1. Contract repairs (decision 1): `--json` on set/heading set,
   env overrides, `error.v1` code field, wire-or-cut dead config,
   retire PLAN.md into the roadmap.
2. Residency substrate (decision 3D): heading-level nodes, agent
   home pages, `orgx context`.
3. Cross-workspace resolution (decision 2): graph edge classes,
   `--all-workspaces`, md nodes in Scan.
4. `orgx serve` read-only (decision 4).
5. `orgx agent` bridge (decision 3C) — after the substrate, if
   headless demand shows up.
6. Embedded agentkit (decision 3B) — trigger-gated, not scheduled.
