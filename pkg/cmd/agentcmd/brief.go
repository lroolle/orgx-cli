package agentcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/roam"
)

// The injected context is budgeted in bytes of trailing whole
// lines: pages are append-only, so the newest lines win, and an
// over-grown flashcards page cannot silently tax every future
// session's context.
const (
	flashcardsBudget = 1200
	journalBudget    = 1500
	journalFileMax   = 2 // most recent daily files considered
)

// buildBrief composes the vault-resident system prompt: identity,
// the graph's current shape, the working contract, and the standing
// context. It reads the vault and writes nothing.
func buildBrief(root, author string) (string, error) {
	g, err := roam.BuildGraph(root)
	if err != nil {
		return "", fmt.Errorf("scan %s: %w", root, err)
	}
	layout := roam.LoadLayout(root)

	var b strings.Builder
	b.WriteString("<orgx-vault>\n")
	fmt.Fprintf(&b, "You are a resident of an orgx vault — a plain-text knowledge\n")
	fmt.Fprintf(&b, "graph (org files, Logseq layout, org-roam node shape) shared by\n")
	fmt.Fprintf(&b, "humans and agents. The vault is the working directory:\n")
	fmt.Fprintf(&b, "  %s\n\n", root)
	fmt.Fprintf(&b, "Layout: %s/ one node per day · %s/ one node per topic\n", layout.Journals, layout.Pages)
	fmt.Fprintf(&b, "Graph now: %d nodes, %d links", len(g.Nodes), len(g.Edges))
	if len(g.Broken) > 0 {
		fmt.Fprintf(&b, ", %d broken", len(g.Broken))
	}
	b.WriteString("\n\n")

	b.WriteString(`Work the graph through orgx (always --json):
  orgx peek <file>           structure, not content
  orgx find "<q>"            search; refs out, not bodies
  orgx get "<ref>"           one section by stable ref
  orgx node new "<title>"    create a page node
  orgx daily "<text>"        journal to today
  orgx graph                 nodes, edges, broken links
  orgx backlinks <id>        who references a node
  orgx id check              ID hygiene after freehand edits

House rules — they keep the graph coherent:
1. Read ` + filepath.ToSlash(filepath.Join(layout.Pages, "flashcards.org")) + ` first: standing facts live there.
2. Prose belongs to your own editor — edit files freely.
   Shape-critical writes go through orgx (daily, capture, node
   new, set, promote) so IDs, LOGBOOK, and timestamps stay right.
3. After freehand edits, verify: orgx id check && orgx graph --json.
4. Link nodes by [[id:<uuid>]], never by file path.
5. Journal completed work: orgx daily "<what>" --as ` + author + ` --yes.
   Facts, not chatter — the human reads that file.
6. Durable facts go to flashcards, sparingly; it loads every session.
7. Reads never write; every orgx write needs --yes.
`)

	if cards := tailBytes(stripFileHead(readFileOr(filepath.Join(layout.PagesDir(root), "flashcards.org"))), flashcardsBudget); cards != "" {
		b.WriteString("\nStanding context (flashcards):\n")
		b.WriteString(cards)
		b.WriteString("\n")
	}
	if journal := recentJournal(layout.JournalsDir(root)); journal != "" {
		b.WriteString("\nRecent journals:\n")
		b.WriteString(journal)
		b.WriteString("\n")
	}
	b.WriteString("</orgx-vault>")
	return b.String(), nil
}

func readFileOr(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

// recentJournal concatenates the newest daily files (oldest first,
// so the story reads forward) under one byte budget.
func recentJournal(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".org") {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) > journalFileMax {
		names = names[:journalFileMax]
	}
	var parts []string
	for i := len(names) - 1; i >= 0; i-- {
		if raw := stripFileHead(readFileOr(filepath.Join(dir, names[i]))); raw != "" {
			parts = append(parts, "-- "+strings.TrimSuffix(names[i], ".org")+" --\n"+raw)
		}
	}
	return tailBytes(strings.Join(parts, "\n"), journalBudget)
}

// stripFileHead drops the file-level preamble — the properties
// drawer and #+ keywords carry node identity, not content, and a
// prompt has no use for them.
func stripFileHead(s string) string {
	lines := strings.Split(s, "\n")
	inDrawer := false
	start := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.EqualFold(trimmed, ":PROPERTIES:"):
			inDrawer = true
		case strings.EqualFold(trimmed, ":END:"):
			inDrawer = false
		case inDrawer, strings.HasPrefix(trimmed, "#+"), trimmed == "":
		default:
			start = i
			return strings.TrimSpace(strings.Join(lines[start:], "\n"))
		}
	}
	return ""
}

// tailBytes keeps the trailing whole lines that fit the budget —
// never a cut in the middle of a line.
func tailBytes(s string, budget int) string {
	s = strings.TrimSpace(s)
	if len(s) <= budget {
		return s
	}
	cut := s[len(s)-budget:]
	if i := strings.IndexByte(cut, '\n'); i >= 0 {
		cut = cut[i+1:]
	}
	return cut
}
