package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var sharedSweepRequests = []struct {
	name string
	path string
	body string
}{
	{"execute", "/api/v1/sweeps/execute", `{"query":"synthetic"}`},
	{"feedback", "/api/v1/discovery/feedback/preview", `{"sweep":{"query":"synthetic"},"proposal_kind":"coverage_gap","candidate_label":"synthetic candidate","language":"en"}`},
}

func sharedSweepTestHandler(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("BCE_DISCOVERY_HMAC_KEY", strings.Repeat("S", 32))
	mux := http.NewServeMux()
	registerSweeps(mux)
	registerFeedbackPreview(mux)
	return mux
}

func assertSweepBusyResponse(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("busy response is not JSON: %v: %s", err, rec.Body.String())
	}
	if rec.Code != http.StatusServiceUnavailable || body.Code != "sweep_busy" || body.State != "not_started" || !strings.Contains(body.Error, "no work started") {
		t.Fatalf("ambiguous busy outcome: HTTP %d %s", rec.Code, rec.Body.String())
	}
}

func TestFeedbackConfigurationFailureIsDistinctFromSweepBusy(t *testing.T) {
	h := sharedSweepTestHandler(t)
	t.Setenv("BCE_DISCOVERY_HMAC_KEY", "")
	tc := sharedSweepRequests[1]
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusServiceUnavailable || body["code"] != "feedback_not_configured" || body["state"] != "not_started" {
		t.Fatalf("configuration failure is not actionable: HTTP %d err=%v body=%s", rec.Code, err, rec.Body.String())
	}
}

func TestSweepAdmissionIsSharedByHTTPRoutesAndDirectEngine(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"items":[]}}}}`))
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	h := sharedSweepTestHandler(t)
	for i := 0; i < cap(activeSweepSlots); i++ {
		activeSweepSlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(activeSweepSlots); i++ {
			<-activeSweepSlots
		}
	}()
	for _, tc := range sharedSweepRequests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
		assertSweepBusyResponse(t, rec)
	}
	if _, err := runSweep(context.Background(), SweepRequest{Query: "synthetic"}); !errors.Is(err, errSweepBusy) {
		t.Fatalf("direct engine bypassed admission: %v", err)
	}
	// Planning is provider-free and remains available when execution is busy.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/sweeps/plan", strings.NewReader(`{"query":"synthetic"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("busy execution blocked planning: HTTP %d %s", rec.Code, rec.Body.String())
	}
	// Invalid input is actionable validation feedback, not a capacity failure.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/sweeps/execute", strings.NewReader(`{"query":""}`)))
	if rec.Code != http.StatusUnprocessableEntity || calls.Load() != 0 {
		t.Fatalf("validation or admission started provider work: HTTP %d calls=%d", rec.Code, calls.Load())
	}
}

func TestSweepRoutesContendAndCancellationReleasesSharedAdmission(t *testing.T) {
	for firstIndex, first := range sharedSweepRequests {
		t.Run(first.name+"_then_other_route", func(t *testing.T) {
			second := sharedSweepRequests[1-firstIndex]
			var calls atomic.Int32
			started := make(chan struct{})
			cancelled := make(chan struct{})
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					close(started)
					<-r.Context().Done()
					close(cancelled)
					return
				}
				_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[]}}}}`))
			}))
			defer upstream.Close()
			t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
			h := sharedSweepTestHandler(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, first.path, strings.NewReader(first.body)).WithContext(ctx)
				h.ServeHTTP(rec, req)
				finished <- rec
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("first route did not reach the synthetic provider")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, second.path, strings.NewReader(second.body)))
			assertSweepBusyResponse(t, rec)
			if calls.Load() != 1 {
				t.Fatalf("competing route started extra work: %d provider calls", calls.Load())
			}
			cancel()
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Fatal("cancelled sweep did not release execution")
			}
			select {
			case <-cancelled:
			case <-time.After(5 * time.Second):
				t.Fatal("caller cancellation did not reach the synthetic provider")
			}
			rec = httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, second.path, strings.NewReader(second.body)))
			if rec.Code != http.StatusOK || calls.Load() != 2 {
				t.Fatalf("next explicit request did not recover: HTTP %d calls=%d body=%s", rec.Code, calls.Load(), rec.Body.String())
			}
		})
	}
}

func TestSharedSweepBudgetPreservesDeadlineAndCallerCancellation(t *testing.T) {
	before := time.Now()
	ctx, release, err := beginSweepExecution(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadlineAt, ok := ctx.Deadline()
	if !ok || deadlineAt.Before(before.Add(sweepExecutionDeadline)) || deadlineAt.After(time.Now().Add(sweepExecutionDeadline)) {
		release()
		t.Fatalf("shared engine did not apply the advertised %v deadline: %v", sweepExecutionDeadline, deadlineAt)
	}
	release()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("releasing admission did not cancel its bounded context")
	}
	parentDeadline := time.Now().Add(time.Second)
	parent, cancel := context.WithDeadline(context.Background(), parentDeadline)
	defer cancel()
	ctx, release, err = beginSweepExecution(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if got, ok := ctx.Deadline(); !ok || !got.Equal(parentDeadline) {
		t.Fatalf("shared sweep extended a caller deadline: %v", got)
	}
	cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("shared sweep ignored caller cancellation")
	}
}

func TestFeedbackSweepSurvivesOrdinaryQueryPreviewDeadline(t *testing.T) {
	// Waiting for the shared Bitmagnet slots is part of the whole-sweep
	// budget. Complete after the old 20s feedback wrapper would have cut off,
	// without delaying any real provider or weakening the source's 8s timeout.
	for i := 0; i < cap(globalBitmagnetSlots); i++ {
		globalBitmagnetSlots <- struct{}{}
	}
	released := false
	releaseProviderSlots := func() {
		if released {
			return
		}
		for i := 0; i < cap(globalBitmagnetSlots); i++ {
			<-globalBitmagnetSlots
		}
		released = true
	}
	defer releaseProviderSlots()
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[]}}}}`)
	h := sharedSweepTestHandler(t)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		tc := sharedSweepRequests[1]
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)).WithContext(ctx))
		finished <- rec
	}()
	select {
	case rec := <-finished:
		t.Fatalf("sweep feedback stopped before its shared budget: HTTP %d %s", rec.Code, rec.Body.String())
	case <-time.After(21 * time.Second):
	}
	releaseProviderSlots()
	select {
	case rec := <-finished:
		var body FeedbackPreviewResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK || body.State != "no_matching_observation_not_concept_absence" {
			t.Fatalf("feedback lost a complete result after 20s: HTTP %d err=%v body=%s", rec.Code, err, rec.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("feedback did not recover after provider capacity became available")
	}
}
