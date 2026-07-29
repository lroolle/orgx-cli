package roam

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// A vault is a directory the layout lives in, marked by a .orgx/
// directory at its root — discoverable by walking up from the
// current directory the way git finds its repository. The layout is
// Logseq's, which earned it: journals for time, pages for topics,
// whiteboards and assets for what is not prose, one flashcards page
// for what should always be in front of you.
//
// The graph is NOT a directory: nodes and links are derived from
// the files by `orgx graph`. Derived data does not live in a vault.

// MarkerDir is the vault marker and config home.
const MarkerDir = ".orgx"

// Layout names the vault subdirectories. Zero values mean defaults;
// a legacy org-roam vault (flat nodes, daily/ journals) is detected
// rather than migrated.
type Layout struct {
	Journals string `yaml:"journals,omitempty"` // default "journals"; "daily" when org-roam's daily/ exists
	Pages    string `yaml:"pages,omitempty"`    // default "pages"
}

// vaultConfig is .orgx/config.yaml. Only layout for now — vault
// config stays small on purpose.
type vaultConfig struct {
	Layout Layout `yaml:"layout,omitempty"`
}

// FindVault walks up from dir looking for the .orgx marker.
func FindVault(dir string) (string, bool) {
	dir = ExpandPath(dir)
	for {
		if info, err := os.Stat(filepath.Join(dir, MarkerDir)); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// LoadLayout resolves the effective layout for a root: vault config
// first, then detection (org-roam's daily/ convention), then
// defaults.
func LoadLayout(root string) Layout {
	layout := Layout{}
	if raw, err := os.ReadFile(filepath.Join(root, MarkerDir, "config.yaml")); err == nil {
		var cfg vaultConfig
		if yaml.Unmarshal(raw, &cfg) == nil {
			layout = cfg.Layout
		}
	}
	if layout.Journals == "" {
		layout.Journals = "journals"
		if dirExists(filepath.Join(root, "daily")) && !dirExists(filepath.Join(root, "journals")) {
			layout.Journals = "daily" // an org-roam-dailies vault; respect it
		}
	}
	if layout.Pages == "" {
		layout.Pages = "pages"
	}
	return layout
}

// JournalsDir and PagesDir are the resolved layout paths.
func (l Layout) JournalsDir(root string) string { return filepath.Join(root, l.Journals) }
func (l Layout) PagesDir(root string) string    { return filepath.Join(root, l.Pages) }

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// InitVault scaffolds the default layout in root. Existing files
// are never touched: init is idempotent and additive.
type InitReport struct {
	Root    string   `json:"root"`
	Created []string `json:"created"`
	Kept    []string `json:"kept"` // already existed
}

func InitVault(root string) (InitReport, error) {
	root = ExpandPath(root)
	report := InitReport{Root: root}

	track := func(rel string, made bool) {
		if made {
			report.Created = append(report.Created, rel)
		} else {
			report.Kept = append(report.Kept, rel)
		}
	}

	for _, dir := range []string{MarkerDir, "journals", "pages", "whiteboards", "assets"} {
		path := filepath.Join(root, dir)
		existed := dirExists(path)
		if !existed {
			if err := os.MkdirAll(path, 0755); err != nil {
				return report, fmt.Errorf("create %s: %w", dir, err)
			}
		}
		track(dir + "/", !existed)
	}

	files := []struct {
		rel     string
		content string
	}{
		{filepath.Join(MarkerDir, "config.yaml"), defaultVaultConfig},
		{filepath.Join("pages", "contents.org"), seedPage("contents",
			"The vault's front door. Link what matters from here.")},
		{filepath.Join("pages", "flashcards.org"), seedPage("flashcards",
			"Durable facts and preferences, one heading each. Agents and\nhumans load this page first — keep it short enough to always read.")},
	}
	for _, f := range files {
		path := filepath.Join(root, f.rel)
		if _, err := os.Stat(path); err == nil {
			track(f.rel, false)
			continue
		}
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			return report, fmt.Errorf("write %s: %w", f.rel, err)
		}
		track(f.rel, true)
	}
	return report, nil
}

const defaultVaultConfig = `# orgx vault config — layout only, and only when it differs from
# the defaults (journals/, pages/).
#layout:
#  journals: journals
#  pages: pages
`

func seedPage(title, body string) string {
	return ":PROPERTIES:\n:ID:       " + NewID() +
		"\n:END:\n#+title: " + title + "\n\n" + body + "\n"
}
