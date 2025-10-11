package utils

import "time"

// Ticker is a small abstraction over a time source that emits ticks.
// Implementations include a real ticker (wrapping time.Ticker) and a fake ticker for tests.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset()
}

// RealTimeTicker wraps time.Ticker
type RealTimeTicker struct {
	t *time.Ticker
	d time.Duration
}

func NewRealTimeTicker(d time.Duration) Ticker {
	return &RealTimeTicker{
		t: time.NewTicker(d),
		d: d,
	}
}

func (r *RealTimeTicker) C() <-chan time.Time { return r.t.C }
func (r *RealTimeTicker) Stop()               { r.t.Stop() }
func (r *RealTimeTicker) Reset()              { r.t.Reset(r.d) }
