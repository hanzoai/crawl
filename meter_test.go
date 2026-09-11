package crawl

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type fakeMeter struct {
	mu                          sync.Mutex
	refuse                      error
	reserved, debited, released int
}

func (m *fakeMeter) Reserve(context.Context) (Charge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.refuse != nil {
		return nil, m.refuse
	}
	m.reserved++
	return fakeCharge{m}, nil
}

type fakeCharge struct{ m *fakeMeter }

func (c fakeCharge) Debit()   { c.m.mu.Lock(); c.m.debited++; c.m.mu.Unlock() }
func (c fakeCharge) Release() { c.m.mu.Lock(); c.m.released++; c.m.mu.Unlock() }

func meterOn(t *testing.T, m Meter) {
	t.Helper()
	prev := meter.Load()
	BindMeter(m)
	t.Cleanup(func() { meter.Store(prev) })
}

var shell = func() *Page { return &Page{URL: "https://example.com/app", Markdown: "Loading…"} }

func TestARenderIsDebitedOnce(t *testing.T) {
	m := &fakeMeter{}
	meterOn(t, m)
	browserAt(t, rendered(strings.Repeat("the rendered article. ", 60)))

	if got := escalate(context.Background(), shell(), "https://example.com/app"); got.Metadata["renderer"] != "browser" {
		t.Fatalf("the thin page was not rendered")
	}
	if m.reserved != 1 || m.debited != 1 || m.released != 1 {
		t.Fatalf("reserved=%d debited=%d released=%d, want 1 1 1", m.reserved, m.debited, m.released)
	}
}

// Failed work is not billed, and the hold still goes back.
func TestAFailedRenderIsNotDebited(t *testing.T) {
	m := &fakeMeter{}
	meterOn(t, m)
	browserAt(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	escalate(context.Background(), shell(), "https://example.com/app")
	if m.debited != 0 || m.released != 1 {
		t.Fatalf("debited=%d released=%d, want 0 1", m.debited, m.released)
	}
}

// A caller who cannot cover the render keeps the page the static read produced,
// and the browser is never asked.
func TestARefusedRenderKeepsTheStaticPage(t *testing.T) {
	meterOn(t, &fakeMeter{refuse: errors.New("insufficient balance")})
	browserAt(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("the browser was asked for a render nobody could pay for")
	})

	static := shell()
	if got := escalate(context.Background(), static, "https://example.com/app"); got != static {
		t.Fatalf("a refused render replaced the static page")
	}
}

// With no renderer bound nothing is reserved, because nothing would be rendered.
func TestNoRendererReservesNothing(t *testing.T) {
	m := &fakeMeter{}
	meterOn(t, m)
	prev := renderer.Load()
	BindRenderer(Renderer{})
	t.Cleanup(func() { renderer.Store(prev) })

	static := shell()
	if got := escalate(context.Background(), static, "https://example.com/app"); got != static || m.reserved != 0 {
		t.Fatalf("rendered=%v reserved=%d with no renderer, want the static page and 0", got != static, m.reserved)
	}
}
