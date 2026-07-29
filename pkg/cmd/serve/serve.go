// Package serve is the vault's read-only web preview: journals,
// pages, backlinks, and the derived graph, rendered per request
// straight from the files. It serves every configured workspace
// plus the vault the shell sits in, and has no write endpoint at
// all — editing belongs to editors and agents, review to git.
package serve

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"time"

	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/roam"
	"github.com/spf13/cobra"
)

//go:embed templates
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

type ServeOptions struct {
	IO *iostreams.IOStreams

	Addr    string
	Vaults  []Vault
	Version string
}

// Vault is one served graph: a name for the URL and its root.
type Vault struct {
	Name string `json:"name"`
	Root string `json:"root"`
}

func NewCmdServe(f *cmdutil.Factory, runF func(*ServeOptions) error) *cobra.Command {
	opts := &ServeOptions{IO: f.IOStreams, Version: f.AppVersion}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Read-only web preview of your vaults",
		Long: `Serve the vaults over HTTP for human reading: journals timeline,
pages with backlinks, rendered org, and the derived graph.

Serves every configured workspace plus the vault the current
directory sits in. Everything is derived from the files on each
request — nothing is cached, so the page is always the truth on
disk. There are no write endpoints: editing belongs to your editor
and your agents, review belongs to git.

Examples:
  orgx serve                        # all workspaces + current vault
  orgx serve -w work                # one workspace only
  orgx serve --addr 127.0.0.1:8080`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.LoadOrDefault()
			ws, _ := cmd.Flags().GetString("workspace")
			rootFlag, _ := cmd.Flags().GetString("root")
			vaults, err := vaultSources(cfg, ws, rootFlag)
			if err != nil {
				return err
			}
			opts.Vaults = vaults
			if runF != nil {
				return runF(opts)
			}
			return serveRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Addr, "addr", "127.0.0.1:6749", "Listen address (6749 spells orgx)")
	return cmd
}

// vaultSources resolves what to serve: an explicit --root or -w
// narrows to one vault; otherwise the union of every configured
// workspace and the vault discovered from the current directory.
func vaultSources(cfg *config.Config, wsName, rootOverride string) ([]Vault, error) {
	if rootOverride != "" {
		root := roam.ExpandPath(rootOverride)
		return []Vault{{Name: filepath.Base(root), Root: root}}, nil
	}
	if wsName != "" {
		ws, err := cfg.GetWorkspace(wsName)
		if err != nil {
			return nil, cmdutil.WithFix(err, "orgx ws list shows configured workspaces")
		}
		return []Vault{{Name: wsName, Root: roam.ExpandPath(ws.Root)}}, nil
	}

	var vaults []Vault
	seenRoot := map[string]bool{}
	seenName := map[string]bool{}
	names := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		root := roam.ExpandPath(cfg.Workspaces[name].Root)
		if seenRoot[root] {
			continue
		}
		seenRoot[root] = true
		seenName[name] = true
		vaults = append(vaults, Vault{Name: name, Root: root})
	}
	if cwd, err := filepath.Abs("."); err == nil {
		if root, ok := roam.FindVault(cwd); ok && !seenRoot[root] {
			name := filepath.Base(root)
			for seenName[name] {
				name += "~"
			}
			vaults = append(vaults, Vault{Name: name, Root: root})
		}
	}
	sort.Slice(vaults, func(i, j int) bool { return vaults[i].Name < vaults[j].Name })

	if len(vaults) == 0 {
		return nil, cmdutil.WithFix(fmt.Errorf("nothing to serve"),
			"orgx init (in your vault), or orgx ws add main --root ~/org/roam, or pass --root")
	}
	return vaults, nil
}

func serveRun(opts *ServeOptions) error {
	s, err := newServer(opts.Vaults, opts.Version)
	if err != nil {
		return err
	}

	label := "vault"
	if len(opts.Vaults) > 1 {
		label = "vaults"
	}
	fmt.Fprintf(opts.IO.Out, "serving %d %s, read-only:\n", len(opts.Vaults), label)
	for _, v := range opts.Vaults {
		nodes, _, _ := roam.Scan(v.Root)
		fmt.Fprintf(opts.IO.Out, "  %-12s %s (%d nodes)\n", v.Name, v.Root, len(nodes))
	}
	fmt.Fprintf(opts.IO.Out, "http://%s\n", opts.Addr)

	srv := &http.Server{
		Addr:              opts.Addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return srv.ListenAndServe()
}

type server struct {
	vaults  map[string]Vault
	order   []Vault
	version string
	tmpl    *template.Template
	mux     *http.ServeMux
}

func newServer(vaults []Vault, version string) (*server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"esc": url.PathEscape,
	}).ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	s := &server{
		vaults:  map[string]Vault{},
		order:   vaults,
		version: version,
		tmpl:    tmpl,
		mux:     http.NewServeMux(),
	}
	for _, v := range vaults {
		s.vaults[v.Name] = v
	}

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	// Method-qualified patterns make the read-only contract
	// structural: anything but GET/HEAD is 405 from the mux itself.
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	s.mux.HandleFunc("GET /{$}", s.home)
	s.mux.HandleFunc("GET /v/{vault}/{$}", s.vaultHome)
	s.mux.HandleFunc("GET /v/{vault}/journals", s.journals)
	s.mux.HandleFunc("GET /v/{vault}/pages", s.pages)
	s.mux.HandleFunc("GET /v/{vault}/node/{id}", s.node)
	s.mux.HandleFunc("GET /v/{vault}/graph", s.graphPage)
	s.mux.HandleFunc("GET /v/{vault}/graph.json", s.graphJSON)
	s.mux.HandleFunc("GET /v/{vault}/assets/{path...}", s.asset)
	return s, nil
}
