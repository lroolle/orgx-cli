package serve

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/roam"
)

// page is the data every template receives; handlers extend it.
type page struct {
	Title   string
	Vault   Vault
	Vaults  []Vault
	Multi   bool
	Version string
	// Base becomes a <base href> on pages that inline rendered org,
	// so the files' relative links (../assets/...) resolve the same
	// everywhere they appear.
	Base string
}

func (s *server) newPage(title string, v Vault) page {
	return page{
		Title:   title,
		Vault:   v,
		Vaults:  s.order,
		Multi:   len(s.order) > 1,
		Version: s.version,
	}
}

func (s *server) render(w http.ResponseWriter, name string, data interface{}) {
	var buf strings.Builder
	if err := s.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(buf.String()))
}

func (s *server) vaultOr404(w http.ResponseWriter, r *http.Request) (Vault, bool) {
	v, ok := s.vaults[r.PathValue("vault")]
	if !ok {
		http.NotFound(w, r)
	}
	return v, ok
}

func nodeBase(v Vault) string {
	return "/v/" + url.PathEscape(v.Name) + "/node/"
}

// home lists the vaults, or goes straight in when there is one —
// a chooser with a single choice is furniture.
func (s *server) home(w http.ResponseWriter, r *http.Request) {
	if len(s.order) == 1 {
		http.Redirect(w, r, "/v/"+url.PathEscape(s.order[0].Name)+"/", http.StatusFound)
		return
	}
	type row struct {
		Vault
		Nodes int
	}
	data := struct {
		page
		Rows []row
	}{page: page{Title: "vaults", Vaults: s.order, Multi: true, Version: s.version}}
	for _, v := range s.order {
		nodes, _, _ := roam.Scan(v.Root)
		data.Rows = append(data.Rows, row{Vault: v, Nodes: len(nodes)})
	}
	s.render(w, "vaults.html", data)
}

type day struct {
	Date  string
	ID    string
	HTML  template.HTML
	Error string
}

// vaultHome is the journals feed — the vault reads like a log,
// newest first, with the pinned pages up top.
func (s *server) vaultHome(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	files := journalFiles(v)
	if len(files) > 7 {
		files = files[:7]
	}
	var days []day
	for _, f := range files {
		d := day{Date: strings.TrimSuffix(filepath.Base(f), ".org")}
		if meta, err := roam.ReadMeta(f); err == nil {
			d.ID = meta.ID
		}
		html, err := renderOrg(f, nodeBase(v))
		if err != nil {
			d.Error = err.Error()
		}
		d.HTML = html
		days = append(days, d)
	}

	nodes, _, _ := roam.Scan(v.Root)
	data := struct {
		page
		Days  []day
		Pins  []roam.NodeMeta
		Nodes int
	}{page: s.newPage(v.Name, v), Days: days, Nodes: len(nodes)}
	data.Base = nodeBase(v)
	for _, n := range nodes {
		title := strings.ToLower(n.Title)
		if title == "contents" || title == "flashcards" {
			data.Pins = append(data.Pins, n)
		}
	}
	sort.Slice(data.Pins, func(i, j int) bool { return data.Pins[i].Title < data.Pins[j].Title })
	s.render(w, "vault.html", data)
}

func (s *server) journals(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	type entry struct {
		Date string
		ID   string
	}
	type month struct {
		Name string
		Days []entry
	}
	var months []month
	for _, f := range journalFiles(v) {
		date := strings.TrimSuffix(filepath.Base(f), ".org")
		e := entry{Date: date}
		if meta, err := roam.ReadMeta(f); err == nil {
			e.ID = meta.ID
		}
		m := date
		if len(date) >= 7 {
			m = date[:7]
		}
		if len(months) == 0 || months[len(months)-1].Name != m {
			months = append(months, month{Name: m})
		}
		months[len(months)-1].Days = append(months[len(months)-1].Days, e)
	}
	data := struct {
		page
		Months []month
	}{page: s.newPage("journals", v), Months: months}
	s.render(w, "journals.html", data)
}

func (s *server) pages(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	nodes, skipped, _ := roam.Scan(v.Root)
	journalsDir := roam.LoadLayout(v.Root).JournalsDir(v.Root)
	var rows []roam.NodeMeta
	for _, n := range nodes {
		if strings.HasPrefix(n.Path, journalsDir+string(filepath.Separator)) {
			continue
		}
		rows = append(rows, n)
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Title) < strings.ToLower(rows[j].Title)
	})
	data := struct {
		page
		Rows    []roam.NodeMeta
		Skipped int
	}{page: s.newPage("pages", v), Rows: rows, Skipped: skipped}
	s.render(w, "pages.html", data)
}

func (s *server) node(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	g, err := roam.BuildGraph(v.Root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n := g.Node(r.PathValue("id"))
	if n == nil {
		http.NotFound(w, r)
		return
	}
	html, renderErr := renderOrg(n.Path, nodeBase(v))
	var broken []roam.Broken
	for _, b := range g.Broken {
		if b.From == n.ID {
			broken = append(broken, b)
		}
	}
	rel, _ := filepath.Rel(v.Root, n.Path)
	data := struct {
		page
		Node      roam.NodeMeta
		Rel       string
		HTML      template.HTML
		Error     string
		Backlinks []roam.NodeMeta
		Broken    []roam.Broken
	}{page: s.newPage(n.Title, v), Node: *n, Rel: rel, HTML: html, Backlinks: g.Backlinks(n.ID), Broken: broken}
	if renderErr != nil {
		data.Error = renderErr.Error()
	}
	data.Base = nodeBase(v)
	s.render(w, "node.html", data)
}

func (s *server) graphPage(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	g, err := roam.BuildGraph(v.Root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		page
		Graph roam.Graph
	}{page: s.newPage("graph", v), Graph: g}
	s.render(w, "graph.html", data)
}

// graphJSON mirrors `orgx graph --json` — same kind, same shape —
// so anything that reads the CLI envelope can read the endpoint.
func (s *server) graphJSON(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	g, err := roam.BuildGraph(v.Root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(struct {
		Kind string `json:"kind"`
		roam.Graph
	}{Kind: "orgx.graph.v1", Graph: g})
}

func (s *server) asset(w http.ResponseWriter, r *http.Request) {
	v, ok := s.vaultOr404(w, r)
	if !ok {
		return
	}
	rel := r.PathValue("path")
	if !filepath.IsLocal(filepath.FromSlash(rel)) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(v.Root, "assets", filepath.FromSlash(rel)))
}

// journalFiles lists the vault's journal org files, newest first
// (the YYYY-MM-DD names make that a reverse sort).
func journalFiles(v Vault) []string {
	dir := roam.LoadLayout(v.Root).JournalsDir(v.Root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".org") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files
}
