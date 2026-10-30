package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func searXNGReservationCount(endpoint string) int {
	searXNGPace.Lock()
	defer searXNGPace.Unlock()
	return len(searXNGPace.Recent[endpoint])
}

func TestSearXNGInvalidInputsDoNotReserveQuotaOrMakeRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	endpoint := server.URL + "/search"
	t.Setenv("BCE_SEARXNG_URL", endpoint)

	cases := []struct {
		name, query, language string
	}{
		{name: "unsafe query", query: "!unsafe", language: "en"},
		{name: "invalid language", query: "fictional", language: "not_a_language"},
	}
	var wg sync.WaitGroup
	for _, tc := range cases {
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := searchSearXNGScoped(context.Background(), tc.query, tc.language); err == nil {
					t.Errorf("accepted invalid %s", tc.name)
				}
			}()
		}
	}
	wg.Wait()
	if got := requests.Load(); got != 0 {
		t.Fatalf("invalid calls made %d SearXNG requests", got)
	}
	if got := searXNGReservationCount(endpoint); got != 0 {
		t.Fatalf("invalid calls reserved %d hourly quota slots", got)
	}
}

func TestSearXNGCanceledCallsDoNotReserveQuotaOrMakeRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	endpoint := server.URL + "/search"
	t.Setenv("BCE_SEARXNG_URL", endpoint)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := searchSearXNGScoped(ctx, "fictional", "en"); err == nil {
		t.Fatal("canceled request unexpectedly succeeded")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("canceled call made %d SearXNG requests", got)
	}
	if got := searXNGReservationCount(endpoint); got != 0 {
		t.Fatalf("canceled call reserved %d hourly quota slots", got)
	}
}

func TestSearXNGCanceledWhilePacingDoesNotReserveQuotaOrMakeRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	endpoint := server.URL + "/search"
	t.Setenv("BCE_SEARXNG_URL", endpoint)
	if _, err := searchSearXNGScoped(context.Background(), "fictional", "en"); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(20*time.Millisecond, cancel)
	defer timer.Stop()
	if _, err := searchSearXNGScoped(ctx, "fictional", "en"); err == nil {
		t.Fatal("request canceled while pacing unexpectedly succeeded")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("paced canceled call made a request; total requests=%d", got)
	}
	if got := searXNGReservationCount(endpoint); got != 1 {
		t.Fatalf("paced canceled call changed hourly reservation count to %d", got)
	}
}
