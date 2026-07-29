// Package roam is the knowledge-graph layer over a workspace of org
// files: file-level nodes (a properties drawer with :ID: plus
// #+title), dailies, and the metadata scan that node listing and
// backlink defaults build on. It reads file heads directly instead
// of full-parsing — listing a thousand-node graph should cost
// milliseconds, not a parser pass per file.
package roam

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lroolle/orgx-cli/pkg/config"
)

// NodeMeta is the file-level identity of one roam node.
type NodeMeta struct {
	Path   string   `json:"path"`
	ID     string   `json:"id,omitempty"`
	Title  string   `json:"title"`
	Tags   []string `json:"tags,omitempty"`
	Author string   `json:"author,omitempty"`
	MTime  string   `json:"mtime"`
}

// ResolveRoot picks the roam root: --root override, then the named
// (or default) workspace. The error states the fix.
func ResolveRoot(cfg *config.Config, wsName, rootOverride string) (string, error) {
	if rootOverride != "" {
		return ExpandPath(rootOverride), nil
	}
	ws, err := cfg.GetWorkspace(wsName)
	if err != nil {
		return "", fmt.Errorf(
			"no roam root: %w — run 'orgx ws add main --root ~/org/roam' then 'orgx ws use main', or pass --root", err)
	}
	return ExpandPath(ws.Root), nil
}

// DailiesDir returns the dailies directory under root for the named
// (or default) workspace; the subdirectory defaults to "daily".
func DailiesDir(cfg *config.Config, wsName, root string) string {
	sub := "daily"
	if ws, err := cfg.GetWorkspace(wsName); err == nil && ws.Dailies != "" {
		sub = ws.Dailies
	}
	return filepath.Join(root, sub)
}

// ExpandPath resolves ~/ and makes relative paths absolute.
func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	if !filepath.IsAbs(p) {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, p)
	}
	return p
}

// Slug converts a title to a filename fragment the org-roam way:
// lowercase, runs of non-alphanumerics collapse to one underscore.
func Slug(title string) string {
	var b strings.Builder
	pending := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if pending && b.Len() > 0 {
				b.WriteByte('_')
			}
			pending = false
			b.WriteRune(r)
		default:
			pending = true
		}
	}
	if b.Len() == 0 {
		return "untitled"
	}
	return b.String()
}

// ReadMeta extracts file-level metadata from the head of an org
// file: the top properties drawer (:ID:, :ORGX_AUTHOR:), #+title,
// and #+filetags. It stops at the first heading — file-level
// keywords cannot appear later.
func ReadMeta(path string) (NodeMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return NodeMeta{}, err
	}
	defer f.Close()

	meta := NodeMeta{Path: path}
	if info, err := f.Stat(); err == nil {
		meta.MTime = info.ModTime().UTC().Format(time.RFC3339)
	}

	inDrawer := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "* "), line == "*":
			// First heading ends the file-level preamble.
			return meta, sc.Err()
		case strings.EqualFold(line, ":PROPERTIES:"):
			inDrawer = true
		case strings.EqualFold(line, ":END:"):
			inDrawer = false
		case inDrawer:
			key, val, ok := strings.Cut(line, ": ")
			if !ok {
				continue
			}
			switch strings.ToUpper(strings.Trim(key, ":")) {
			case "ID":
				meta.ID = strings.TrimSpace(val)
			case "ORGX_AUTHOR":
				meta.Author = strings.TrimSpace(val)
			}
		case strings.HasPrefix(strings.ToLower(line), "#+title:"):
			meta.Title = strings.TrimSpace(line[len("#+title:"):])
		case strings.HasPrefix(strings.ToLower(line), "#+filetags:"):
			meta.Tags = parseFiletags(line[len("#+filetags:"):])
		}
	}
	return meta, sc.Err()
}

func parseFiletags(s string) []string {
	var tags []string
	for _, t := range strings.Split(strings.TrimSpace(s), ":") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// Scan lists the roam nodes under root: org files whose head carries
// an :ID:. Files without one are notes but not nodes; they are
// counted so callers can say so instead of silently dropping them.
func Scan(root string) (nodes []NodeMeta, skipped int, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".org" {
			return nil
		}
		meta, err := ReadMeta(path)
		if err != nil {
			skipped++
			return nil
		}
		if meta.ID == "" {
			skipped++
			return nil
		}
		if meta.Title == "" {
			meta.Title = strings.TrimSuffix(filepath.Base(path), ".org")
		}
		nodes = append(nodes, meta)
		return nil
	})
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })
	return nodes, skipped, err
}
