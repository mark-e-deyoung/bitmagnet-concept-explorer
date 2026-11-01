package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func productionSweepTestServer() *httptest.Server {
	return productionSweepTestServerWithWriteTimeout(sweepHTTPWriteTimeout)
}

func productionSweepTestServerWithWriteTimeout(writeTimeout time.Duration) *httptest.Server {
	srv := httptest.NewUnstartedServer(authenticated(handler(), "", ""))
	srv.Config.ReadHeaderTimeout = 5 * time.Second
	srv.Config.ReadTimeout = 15 * time.Second
	srv.Config.WriteTimeout = writeTimeout
	srv.Config.IdleTimeout = 30 * time.Second
	srv.Start()
	return srv
}

func TestProductionSweepDeadlineBelowAndAboveOldBoundary(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[]}}}}`)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	var seen int
	var delay time.Duration
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		time.Sleep(delay)
		_, _ = io.WriteString(w, `{"results":[],"suggestions":[]}`)
	}))
	defer search.Close()
	t.Setenv("BCE_SEARXNG_URL", search.URL+"/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	srv := productionSweepTestServer()
	defer srv.Close()
	payload := SweepRequest{Query: "animation", ConceptIDs: []string{"media.animation"}, Languages: []string{"en", "de", "th"}, IncludeSearxng: true, ExternalConsent: true, MaxSearxngQueries: 3}
	body, _ := json.Marshal(payload)
	client := http.Client{Timeout: sweepClientTimeout}
	call := func(wantState string, min, max time.Duration) {
		t.Helper()
		start := time.Now()
		resp, err := client.Post(srv.URL+"/api/v1/sweeps/execute", "application/json", bytes.NewReader(body))
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("sweep transport failed at %v: %v", elapsed, err)
		}
		defer resp.Body.Close()
		var got SweepResult
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode result: %v", err)
		}
		if resp.StatusCode != http.StatusOK || got.State != wantState || len(got.External) != 3 || got.DeadlineSeconds != 55 {
			t.Fatalf("status=%d state=%s external=%d deadline=%d", resp.StatusCode, got.State, len(got.External), got.DeadlineSeconds)
		}
		if elapsed < min || elapsed > max {
			t.Fatalf("elapsed %v outside [%v,%v]", elapsed, min, max)
		}
		t.Logf("production-equivalent API result: state=%s elapsed=%s external=%d", got.State, elapsed, len(got.External))
	}
	call("complete", 0, 15*time.Second)
	// Exact production timeout configuration used to return EOF here because
	// three 5.4s synthetic responses finish after its old 15s WriteTimeout.
	delay = 5400 * time.Millisecond
	search.Close()
	search = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		time.Sleep(delay)
		_, _ = io.WriteString(w, `{"results":[],"suggestions":[]}`)
	}))
	defer search.Close()
	t.Setenv("BCE_SEARXNG_URL", search.URL+"/search")
	call("complete", 15*time.Second, 25*time.Second)
	if seen != 6 {
		t.Fatalf("expected one execution per request and exactly six provider requests, got %d", seen)
	}
	if sweepClientTimeout <= sweepHTTPWriteTimeout || sweepExecutionDeadline >= sweepHTTPWriteTimeout {
		t.Fatal("CLI/server/deadline contract is not ordered")
	}
	if cap(activeSweepSlots) != maxConcurrentSweeps || maxConcurrentSweeps != 1 {
		t.Fatalf("sweep concurrency bound=%d", cap(activeSweepSlots))
	}
}

func TestLegacyWriteTimeoutReproducesMissingSweepResponse(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"torrentContent":{"search":{"items":[]}}}}`)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	var requests atomic.Int32
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(5400 * time.Millisecond)
		_, _ = io.WriteString(w, `{"results":[],"suggestions":[]}`)
	}))
	defer search.Close()
	t.Setenv("BCE_SEARXNG_URL", search.URL+"/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	srv := productionSweepTestServerWithWriteTimeout(15 * time.Second)
	defer srv.Close()
	body, _ := json.Marshal(SweepRequest{Query: "animation", ConceptIDs: []string{"media.animation"}, Languages: []string{"en", "de", "th"}, IncludeSearxng: true, ExternalConsent: true, MaxSearxngQueries: 3})
	start := time.Now()
	resp, err := (&http.Client{Timeout: 25 * time.Second}).Post(srv.URL+"/api/v1/sweeps/execute", "application/json", bytes.NewReader(body))
	elapsed := time.Since(start)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil || elapsed < 15*time.Second || elapsed > 25*time.Second || requests.Load() != 3 {
		t.Fatalf("legacy 15s WriteTimeout failure not reproduced: elapsed=%v err=%v requests=%d", elapsed, err, requests.Load())
	}
	t.Logf("legacy production-equivalent API result: elapsed=%s error=%v provider_requests=%d", elapsed, err, requests.Load())
}

func TestLegacyWriteTimeoutIncludesSlowRequestBodyAndLosesCompletedResult(t *testing.T) {
	// WriteTimeout begins when request headers arrive, so time spent reading the
	// body consumes the same allowance as handler execution and response writing.
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			return
		}
		time.Sleep(55 * time.Second)
		_, _ = io.WriteString(w, `{"state":"complete"}`)
	}))
	srv.Config.ReadHeaderTimeout = 5 * time.Second
	srv.Config.ReadTimeout = 15 * time.Second
	srv.Config.WriteTimeout = 60 * time.Second
	srv.Start()
	defer srv.Close()

	reader, writer := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, srv.URL, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	go func() {
		_, _ = io.WriteString(writer, `{"x":`)
		time.Sleep(6 * time.Second)
		_, _ = io.WriteString(writer, `1}`)
		_ = writer.Close()
	}()
	started := time.Now()
	resp, err := (&http.Client{Timeout: 75 * time.Second}).Do(req)
	elapsed := time.Since(started)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil || elapsed < 60*time.Second || elapsed > 70*time.Second {
		t.Fatalf("legacy 60s write budget should lose completed 55s handler after slow body: elapsed=%v err=%v", elapsed, err)
	}
	t.Logf("legacy slow-body result lost: body_delay=6s handler=55s elapsed=%s error=%v", elapsed, err)
}

func TestSweepAdmissionRejectsWithoutStartingWork(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{"data":{"torrentContent":{"search":{"items":[]}}}}`)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	for i := 0; i < cap(activeSweepSlots); i++ {
		activeSweepSlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(activeSweepSlots); i++ {
			<-activeSweepSlots
		}
	}()
	srv := productionSweepTestServer()
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/v1/sweeps/execute", "application/json", bytes.NewBufferString(`{"query":"synthetic"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusServiceUnavailable || !bytes.Contains(b, []byte("no work started")) || calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", resp.StatusCode, calls, b)
	}
}

func TestSweepCancellationStopsDownstreamAndDoesNotRetry(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"torrentContent":{"search":{"items":[]}}}}`)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	canceled := make(chan struct{}, 1)
	var requests int32
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer search.Close()
	t.Setenv("BCE_SEARXNG_URL", search.URL+"/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	srv := productionSweepTestServer()
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	body := bytes.NewBufferString(`{"query":"synthetic","include_searxng":true,"external_consent":true}`)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/api/v1/sweeps/execute", body)
	req.Header.Set("Content-Type", "application/json")
	_, err := http.DefaultClient.Do(req)
	if err == nil {
		t.Fatal("canceled caller unexpectedly received a response")
	}
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("request cancellation did not reach synthetic SearXNG")
	}
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&requests) != 1 {
		t.Fatalf("automatic retry or duplicate request observed: %d", requests)
	}
}

func TestSweepPartialStateRemainsExplicit(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"torrentContent":{"search":{"items":[{"title":"synthetic","contentType":"movie","torrent":{"infoHash":"abc123"}}]}}}}`)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "synthetic refusal", http.StatusServiceUnavailable)
	}))
	defer search.Close()
	t.Setenv("BCE_SEARXNG_URL", search.URL+"/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	srv := productionSweepTestServer()
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/v1/sweeps/execute", "application/json", bytes.NewBufferString(`{"query":"synthetic","include_searxng":true,"external_consent":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var got SweepResult
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || got.State != "partial" || got.Bitmagnet == nil || len(got.Failures) == 0 || !strings.Contains(got.Plan.Warnings[1], "never establish") {
		t.Fatalf("partial result contract lost: status=%d result=%#v", resp.StatusCode, got)
	}
}

func TestSweepCLIEmitsSameJSONOutcome(t *testing.T) {
	if sweepClientTimeout <= sweepHTTPWriteTimeout {
		t.Fatal("CLI timeout must exceed server write timeout")
	}
	setupFixtureGraph(t, sweepFixture)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5400 * time.Millisecond)
		_, _ = io.WriteString(w, `{"data":{"torrentContent":{"search":{"items":[]}}}}`)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	var externalRequests atomic.Int32
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalRequests.Add(1)
		time.Sleep(7200 * time.Millisecond)
		_, _ = io.WriteString(w, `{"results":[],"suggestions":[]}`)
	}))
	defer search.Close()
	t.Setenv("BCE_SEARXNG_URL", search.URL+"/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	listener, err := net.Listen("tcp", "127.0.0.1:3086")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: authenticated(handler(), "", ""), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: sweepHTTPWriteTimeout, IdleTimeout: 30 * time.Second}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()
	bin := filepath.Join(t.TempDir(), "bce")
	build := exec.Command("go", "build", "-o", bin, "./cmd/bce")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v: %s", err, out)
	}
	cmd := exec.Command(bin, "-operation", "sweep", "-server", "http://127.0.0.1:3086", "-input", "-")
	cmd.Stdin = strings.NewReader(`{"query":"animation","concept_ids":["media.animation"],"languages":["en","de","th"],"include_searxng":true,"external_consent":true,"max_searxng_queries":3}`)
	cmd.Env = append(os.Environ(), "BCE_AUTH_USER=", "BCE_AUTH_PASSWORD=")
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("CLI failed: %v: %s", err, out)
	}
	t.Logf("production-equivalent CLI full result: elapsed=%s", elapsed)
	var got SweepResult
	if err := json.Unmarshal(out, &got); err != nil || got.State != "complete" || len(got.External) != 3 || elapsed <= 25*time.Second || elapsed >= 45*time.Second || externalRequests.Load() != 3 {
		t.Fatalf("CLI did not preserve API JSON outcome: err=%v state=%s output=%s", err, got.State, out)
	}
	// The next CLI request receives an explicit partial JSON result because the
	// endpoint's process-local quota has been exhausted; no automatic retry or
	// extra provider request is allowed.
	cmd = exec.Command(bin, "-operation", "sweep", "-server", "http://127.0.0.1:3086", "-input", "-")
	cmd.Stdin = strings.NewReader(`{"query":"animation","concept_ids":["media.animation"],"languages":["en","de","th"],"include_searxng":true,"external_consent":true,"max_searxng_queries":3}`)
	cmd.Env = append(os.Environ(), "BCE_AUTH_USER=", "BCE_AUTH_PASSWORD=")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI partial response failed: %v: %s", err, out)
	}
	if err := json.Unmarshal(out, &got); err != nil || got.State != "partial" || len(got.Failures) == 0 || externalRequests.Load() != 3 {
		t.Fatalf("CLI did not preserve explicit partial outcome without a quota bypass: err=%v state=%s requests=%d output=%s", err, got.State, externalRequests.Load(), out)
	}
}
