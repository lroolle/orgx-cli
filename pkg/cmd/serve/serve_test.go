package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/orgx-cli/pkg/config"
	"github.com/lroolle/orgx-cli/pkg/roam"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func testVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := roam.InitVault(root); err != nil {
		t.Fatal(err)
	}
	write(t, root, "pages/alpha.org",
		":PROPERTIES:\n:ID: id-alpha\n:END:\n#+title: Alpha\n\nSee [[id:id-beta][Beta]] and [[id:ghost][gone]].\n")
	write(t, root, "pages/beta.org",
		":PROPERTIES:\n:ID: id-beta\n:END:\n#+title: Beta\n\nBody of beta.\n")
	write(t, root, "journals/2026-07-29.org",
		":PROPERTIES:\n:ID: id-day\n:END:\n#+title: 2026-07-29\n\n* 10:00 touched [[id:id-alpha][Alpha]]  :@claude:\n")
	return root
}

func testServer(t *testing.T, vaults ...Vault) *httptest.Server {
	t.Helper()
	s, err := newServer(vaults, "test")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.mux)
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, ts *httptest.Server, path string) (int, string) {
	t.Helper()
	res, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var b strings.Builder
	buf := make([]byte, 32*1024)
	for {
		n, err := res.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return res.StatusCode, b.String()
}

func TestVaultSourcesUnionAndDedupe(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	cfg := &config.Config{Workspaces: map[string]config.Workspace{
		"work": {Root: rootA},
		"same": {Root: rootA}, // duplicate root: served once
		"home": {Root: rootB},
	}}
	vaults, err := vaultSources(cfg, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(vaults) != 2 {
		t.Fatalf("want 2 vaults after dedupe, got %v", vaults)
	}
	// The alphabetically-first name wins a duplicated root.
	if vaults[0].Name != "home" || vaults[1].Name != "same" {
		t.Fatalf("want name-sorted [home same], got %v", vaults)
	}
}

func TestVaultSourcesEmptyErrorsWithFix(t *testing.T) {
	if _, err := vaultSources(&config.Config{Workspaces: map[string]config.Workspace{}}, "", ""); err == nil {
		t.Fatal("want an error when nothing resolves")
	}
}

func TestNodePageRendersContentAndBacklinks(t *testing.T) {
	root := testVault(t)
	ts := testServer(t, Vault{Name: "main", Root: root})

	code, body := get(t, ts, "/v/main/node/id-alpha")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	for _, want := range []string{
		"Alpha",
		`href="/v/main/node/id-beta"`, // [[id:...]] rewritten to preview URLs
		"backlinks",
		"2026-07-29",   // the journal links here
		"[[id:ghost]]", // broken outgoing link surfaced
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("node page missing %q:\n%s", want, body)
		}
	}
}

func TestVaultHomeShowsJournalsFeed(t *testing.T) {
	root := testVault(t)
	ts := testServer(t, Vault{Name: "main", Root: root})

	code, body := get(t, ts, "/v/main/")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	if !strings.Contains(body, "2026-07-29") || !strings.Contains(body, "touched") {
		t.Fatalf("journals feed missing:\n%s", body)
	}
	if !strings.Contains(body, "flashcards") {
		t.Fatalf("pinned pages missing:\n%s", body)
	}
}

func TestGraphJSONMirrorsCLIEnvelope(t *testing.T) {
	root := testVault(t)
	ts := testServer(t, Vault{Name: "main", Root: root})

	code, body := get(t, ts, "/v/main/graph.json")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	for _, want := range []string{`"kind": "orgx.graph.v1"`, `"id-alpha"`, `"ghost"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("graph.json missing %q:\n%s", want, body)
		}
	}
}

func TestHomeRedirectsSingleVaultAndListsMany(t *testing.T) {
	root := testVault(t)
	single := testServer(t, Vault{Name: "main", Root: root})
	res, err := http.Get(single.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if got := res.Request.URL.Path; got != "/v/main/" {
		t.Fatalf("single vault should land on /v/main/, got %s", got)
	}

	other := testVault(t)
	multi := testServer(t, Vault{Name: "a", Root: root}, Vault{Name: "b", Root: other})
	code, body := get(t, multi, "/")
	if code != 200 || !strings.Contains(body, `href="/v/a/"`) || !strings.Contains(body, `href="/v/b/"`) {
		t.Fatalf("multi-vault home wrong (%d):\n%s", code, body)
	}
}

func TestReadOnlyIsStructural(t *testing.T) {
	root := testVault(t)
	ts := testServer(t, Vault{Name: "main", Root: root})

	for _, path := range []string{"/v/main/", "/v/main/node/id-alpha", "/v/main/graph.json"} {
		res, err := http.Post(ts.URL+path, "text/plain", strings.NewReader("x"))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s: want 405, got %d", path, res.StatusCode)
		}
	}
}

func TestAssetTraversalIsRejected(t *testing.T) {
	root := testVault(t)
	write(t, root, "assets/pic.txt", "ok")
	write(t, root, "secret.txt", "no")
	ts := testServer(t, Vault{Name: "main", Root: root})

	if code, body := get(t, ts, "/v/main/assets/pic.txt"); code != 200 || body != "ok" {
		t.Fatalf("asset should serve, got %d %q", code, body)
	}
	// Encoded traversal reaches the handler as a literal ".." segment.
	if code, _ := get(t, ts, "/v/main/assets/%2e%2e/secret.txt"); code == 200 {
		t.Fatal("traversal must not serve files outside assets/")
	}
}

func TestUnknownVaultIs404(t *testing.T) {
	root := testVault(t)
	ts := testServer(t, Vault{Name: "main", Root: root})
	if code, _ := get(t, ts, "/v/nope/"); code != 404 {
		t.Fatalf("want 404 for unknown vault, got %d", code)
	}
}
