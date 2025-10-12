package utils

import (
	"testing"
	"time"
)

func TestRealTimeTicker_Smoke(t *testing.T) {
	// use a short duration so the test is quick
	rt := NewRealTimeTicker(10 * time.Millisecond)
	defer rt.Stop()

	select {
	case <-rt.C():
		// success: we received a tick
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for real ticker tick")
	}
}
