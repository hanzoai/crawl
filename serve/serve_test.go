package serve

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hanzoai/crawl"
	"github.com/zap-proto/zip"
)

func mount(t *testing.T, o Options) *zip.App {
	t.Helper()
	app := zip.New(zip.Config{AppName: "crawl", DisableStartupMessage: true})
	if err := Mount(app, o); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	return app
}

func post(t *testing.T, app *zip.App, path, body string) (int, string) {
	t.Helper()
	rq := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(rq)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(b))
}

// pages is an archive that answers every key with one page and records the keys
// it was asked for.
type pages struct {
	mu   sync.Mutex
	gets []string
	md   string
}

func (p *pages) Get(_ context.Context, key string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gets = append(p.gets, key)
	if p.md == "" {
		return nil, errors.New("not found")
	}
	return json.Marshal(map[string]any{"url": "https://example.com/x", "title": "X", "markdown": p.md})
}

func (p *pages) Put(context.Context, string, []byte) error { return nil }

func bindPages(t *testing.T, p *pages) {
	t.Helper()
	crawl.Bind(p)
	t.Cleanup(func() { crawl.Bind(nil) })
}

const refusal = `{"success":false,"error":"missing url"}`

// An empty url, a body that is not JSON and a body past the cap are one refusal,
// in this API's own body.
func TestAMissingURLIsTheDomainRefusal(t *testing.T) {
	app := mount(t, Options{})
	big := `{"url":"https://example.com","pad":"` + strings.Repeat("x", maxRequest) + `"}`
	for _, body := range []string{`{"url":""}`, `not json at all`, `{`, big} {
		if code, out := post(t, app, Path, body); code != http.StatusBadRequest || out != refusal {
			t.Errorf("POST %.40q = %d %s, want 400 %s", body, code, out, refusal)
		}
	}
}

func TestServedAtOneAddress(t *testing.T) {
	app := mount(t, Options{})
	var paths []string
	for _, r := range app.Fiber().GetRoutes() {
		if r.Method == http.MethodPost && strings.Contains(r.Path, "crawl") {
			paths = append(paths, r.Path)
		}
	}
	if len(paths) != 1 || paths[0] != Path {
		t.Fatalf("router has POST %v, want exactly [%s]", paths, Path)
	}
	for _, p := range []string{Path, Path + "/"} {
		if code, _ := post(t, app, p, `{"url":""}`); code != http.StatusBadRequest {
			t.Errorf("POST %s = %d, want 400", p, code)
		}
	}
}

func TestAdmitRefusesBeforeTheFetcher(t *testing.T) {
	p := &pages{md: "never read"}
	bindPages(t, p)
	app := mount(t, Options{Admit: func(*zip.Ctx) bool { return false }})

	code, out := post(t, app, Path, `{"url":"https://example.com/x"}`)
	if code != http.StatusUnauthorized || !strings.Contains(out, `"success":false`) {
		t.Fatalf("an unadmitted crawl = %d %s, want 401", code, out)
	}
	if len(p.gets) != 0 {
		t.Fatalf("an unadmitted caller reached the archive %d time(s)", len(p.gets))
	}
}

// Scope is the decision every entry reaches, so an MCP call — which runs no
// middleware — is refused by it too.
func TestScopeRefusesAToolCall(t *testing.T) {
	p := &pages{md: "never read"}
	bindPages(t, p)
	app := mount(t, Options{Scope: func(context.Context) (crawl.Scope, error) {
		return crawl.Scope{}, zip.ErrUnauthorized("no principal")
	}})

	rq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_page","arguments":{"url":"https://example.com/x"}}}`))
	rq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(rq)
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(raw), `"success":true`) {
		t.Fatalf("a refused scope still read the page: %s", raw)
	}
	if len(p.gets) != 0 {
		t.Fatalf("a refused scope reached the archive %d time(s)", len(p.gets))
	}
}

// The scope a host verifies is the archive key the page is read under.
func TestThePageIsReadUnderTheCallersScope(t *testing.T) {
	p := &pages{md: "an archived article"}
	bindPages(t, p)
	app := mount(t, Options{Scope: func(context.Context) (crawl.Scope, error) {
		return crawl.Scope{Org: "acme"}, nil
	}})

	code, out := post(t, app, Path, `{"url":"https://example.com/x"}`)
	if code != http.StatusOK || !strings.Contains(out, `"markdown":"an archived article"`) {
		t.Fatalf("POST = %d %s, want the archived page", code, out)
	}
	if len(p.gets) != 1 || !strings.HasPrefix(p.gets[0], "crawl/acme-") {
		t.Fatalf("archive keys %v, want one under crawl/acme-", p.gets)
	}
}

func TestReadPageIsATool(t *testing.T) {
	app := mount(t, Options{})
	rq := httptest.NewRequest(http.MethodPost, "/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	rq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(rq)
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("POST /mcp did not answer MCP: %.200s", raw)
	}
	for _, tool := range env.Result.Tools {
		if tool.Name == "read_page" {
			if tool.Description == "" {
				t.Fatal("read_page has no description; run go generate ./serve")
			}
			return
		}
	}
	t.Fatalf("read_page is not a tool: %.300s", raw)
}
