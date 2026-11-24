package main

import "testing"

// Every httptest localhost port is recycled by the OS after a test closes its
// listener. Production SearXNG pacing is intentionally process-wide and keyed
// by endpoint URL; an unrelated later test can accidentally inherit exhausted
// hourly quota or a 403/429 cooldown when the OS reuses that exact port.
// Never weaken production pacing to make such tests pass. Instead isolate only
// the caller's synthetic endpoint, before and after its fixture executes.
func isolateSyntheticSearXNGPace(t *testing.T, endpoint string) {
	t.Helper()
	clear := func() {
		searXNGPace.Lock()
		delete(searXNGPace.Next, endpoint)
		delete(searXNGPace.Recent, endpoint)
		delete(searXNGPace.BlockedUntil, endpoint)
		searXNGPace.Unlock()
	}
	clear()
	t.Cleanup(clear)
}
