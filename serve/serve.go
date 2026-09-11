// Package serve is the crawl engine as an API: POST /v1/crawl on a zip app,
// which is also an MCP tool (read_page), a CLI command and an OpenAPI operation.
package serve

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/hanzoai/crawl"
	"github.com/zap-proto/zip"
)

//go:generate go run github.com/zap-proto/zip/cmd/zipdoc

// Path is where the API answers.
const Path = "/v1/crawl"

// Options is what a host decides about who may crawl.
type Options struct {
	// Admit reports whether an HTTP request may reach the fetcher, and answers 401
	// when it may not. The fetcher dials an address the caller chose, so behind a
	// public address this must refuse an unauthenticated caller. Nil admits
	// everyone, which is right only on loopback.
	Admit func(*zip.Ctx) bool
	// Scope says whose archive a call reads and writes, from the caller's verified
	// identity and never from the body. It is asked on EVERY entry — HTTP, MCP and
	// CLI — because the last two reach the operation with no middleware. An error
	// refuses the call. Nil files every page in the shared archive.
	Scope func(context.Context) (crawl.Scope, error)
}

// crawlRequest is the /v1/crawl body. One URL per call: batching would make the
// response a partial-failure envelope every caller then has to unpack.
type crawlRequest struct {
	// URL is the page to read, absolute and http or https — no other scheme is
	// dialled. An address that turns out to be loopback, link-local, private or
	// multicast is refused at the dialer, redirects included. Empty is answered
	// with success false and the reason in error.
	URL string `json:"url"`
}

// crawlResult is the /v1/crawl response.
//
// Success is a field rather than an HTTP status because "the page could not be
// fetched" is a normal outcome of asking about a URL, not a fault of the request.
type crawlResult struct {
	// Success is whether the page was fetched and read. FALSE with an Error is a
	// complete answer, not a fault — check this before reading Data.
	Success bool `json:"success"`
	// Data is the page, present exactly when Success.
	Data *crawlDocument `json:"data,omitempty"`
	// Error says what stopped the fetch: the host was refused, unreachable, or
	// served something that is not a document.
	Error string `json:"error,omitempty"`
}

// StatusCode states the one non-2xx this answer carries in its own body: a
// request with no url is 400 `{"success":false,"error":"missing url"}`. Every
// other outcome, a failed fetch included, is 200.
func (r *crawlResult) StatusCode() int {
	if !r.Success && r.Error == missingURL {
		return http.StatusBadRequest
	}
	return http.StatusOK
}

const missingURL = "missing url"

// crawlDocument is the crawled page.
type crawlDocument struct {
	// URL is the address actually read, after redirects.
	URL string `json:"url"`
	// Title is the document's title, when it carried one.
	Title string `json:"title,omitempty"`
	// Markdown is the page's content, extracted and rendered to markdown. This is
	// the field to read.
	Markdown string `json:"markdown"`
	// Metadata is whatever the document said about itself — description, og:*,
	// language — plus the response status, the final URL and the content type. It
	// is an open key space, carried as raw JSON so the document publishes "any
	// JSON", which is true.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// maxRequest is what this route reads from a caller. One URL does not need more.
const maxRequest = 1 << 20

// Mount registers POST /v1/crawl on app.
func Mount(app *zip.App, o Options) error {
	if app == nil {
		return errors.New("crawl: nil app")
	}
	// The raw-body facts a typed op cannot see, asked where the bytes still are: a
	// body past the cap or not JSON is the same 400 as a missing url, and it is
	// refused before admission is asked.
	app.Use(zip.H(func(c *zip.Ctx) error {
		if route(c.Path()) != Path {
			return c.Continue()
		}
		if b := c.Body(); len(b) > maxRequest || (len(b) > 0 && !json.Valid(b)) {
			return c.JSON(http.StatusBadRequest, crawlResult{Error: missingURL})
		}
		if o.Admit != nil && !o.Admit(c) {
			return c.JSON(http.StatusUnauthorized, crawlResult{Error: "crawling requires a validated principal"})
		}
		return c.Continue()
	}))
	h := handler{scope: o.Scope}
	zip.Post(app, Path, h.read,
		zip.WithOperationID("read_page"),
		zip.WithSummary("Fetch one URL and read it back as markdown"),
		zip.WithStatus(http.StatusOK, http.StatusBadRequest))
	return nil
}

// route is the path the router matched: it matches case-insensitively and
// ignores a trailing slash, so the raw spelling cannot be compared.
func route(p string) string {
	p = strings.ToLower(p)
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}

type handler struct {
	scope func(context.Context) (crawl.Scope, error)
}

// read fetches one URL and answers with the page as markdown.
//
// It answers with the address it actually landed on, the document's title, its
// content rendered to MARKDOWN, and whatever the page said about itself.
//
// A PAGE THAT COULD NOT BE FETCHED IS A NORMAL ANSWER, not a fault. An
// unreachable host, a refused address and a content type that is not a document
// all answer 200 with `success:false` and the reason in `error`. Non-2xx is
// reserved for a caller problem — 400 with the same body when there is no url,
// 401 when the caller is not admitted — so error handling can trust the status.
// Check `success` before reading `data`.
//
// Pages are kept under the caller's verified scope, never a scope named in the
// body, so a URL already read under that scope is answered without touching the
// network.
//
// The URL is caller-supplied, which makes this a request-forgery primitive by
// construction. Only http and https are accepted, and every address actually
// dialled must be public unicast — loopback, link-local, private and multicast
// are refused. The check lives in the DIALER rather than on the hostname, because
// resolving a name to validate it and then letting the transport resolve it again
// is a gap DNS rebinding walks straight through; redirects re-enter the same
// dialer.
func (h handler) read(ctx context.Context, in *crawlRequest) (*crawlResult, error) {
	var s crawl.Scope
	if h.scope != nil {
		var err error
		if s, err = h.scope(ctx); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(in.URL) == "" {
		return &crawlResult{Error: missingURL}, nil
	}
	page, err := crawl.Read(ctx, s, in.URL)
	if err != nil {
		return &crawlResult{Error: err.Error()}, nil
	}
	meta, err := json.Marshal(page.Metadata)
	if err != nil {
		return nil, zip.ErrInternal("crawl: the page's metadata will not encode")
	}
	return &crawlResult{
		Success: true,
		Data:    &crawlDocument{URL: page.URL, Title: page.Title, Markdown: page.Markdown, Metadata: meta},
	}, nil
}
