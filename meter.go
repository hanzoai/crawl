package crawl

import (
	"context"
	"sync/atomic"
)

// Meter pays for a render.
//
// Fetching a page is one HTTP GET and costs nothing worth metering. Rendering one
// boots a browser, runs the page's scripts and holds it for seconds, so a host
// that sells crawling prices the render and only the render.
//
// Reserve is asked BEFORE the browser is, so a caller who cannot cover a render
// is refused having spent nothing — and the refusal only costs the render: the
// static page still answers. The Charge is debited once a page came back and
// released on every exit, so failed work is never billed.
type Meter interface {
	Reserve(ctx context.Context) (Charge, error)
}

// Charge is one reserved render.
type Charge interface {
	// Debit bills the render. It is called at most once, after the render succeeded.
	Debit()
	// Release returns whatever Debit did not take. It is always called.
	Release()
}

var meter atomic.Pointer[Meter]

// BindMeter installs the process-wide meter. A nil meter renders for free.
func BindMeter(m Meter) {
	if m == nil {
		meter.Store(nil)
		return
	}
	meter.Store(&m)
}

// reserve asks the bound meter, or reserves nothing when there is none.
func reserve(ctx context.Context) (Charge, error) {
	m := meter.Load()
	if m == nil {
		return free{}, nil
	}
	return (*m).Reserve(ctx)
}

// free is the charge a host with no meter holds.
type free struct{}

func (free) Debit()   {}
func (free) Release() {}
