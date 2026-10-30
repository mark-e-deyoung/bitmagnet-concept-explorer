package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSweepPlanAPIReadOnlyWithoutGraphOrNetwork(t *testing.T) {
	service := httptest.NewServer(handler())
	defer service.Close()
	body := `{"query":"fictional title","languages":["en","de"],"max_bitmagnet_queries":2}`
	res, e := http.Post(service.URL+"/api/v1/sweeps/plan", "application/json", bytes.NewBufferString(body))
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	var out SweepPlan
	if e := json.NewDecoder(res.Body).Decode(&out); e != nil || res.StatusCode != 200 || !out.ReadOnly || len(out.Bitmagnet) != 1 || out.ExternalAdmitted {
		t.Fatalf("incorrect safe one-shot plan %#v %v, HTTP %d", out, e, res.StatusCode)
	}
}
func TestSweepLimitsNoBackendAddressOrSecretsExposed(t *testing.T) {
	t.Setenv("BCE_SEARXNG_URL", "http://127.0.0.1:8081/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	rr := httptest.NewRecorder()
	handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/sweeps/limits", nil))
	if rr.Code != 200 {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var limit SweepLimits
	if e := json.Unmarshal(rr.Body.Bytes(), &limit); e != nil {
		t.Fatal(e)
	}
	if limit.MaxSearxngJSONRequestsPerHour != 3 || limit.MaxSearxngQueriesPerSweep != 3 || !limit.SearxngSweepAdmitted {
		t.Fatalf("unexpected limits %#v", limit)
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("127.0.0.1")) || bytes.Contains(rr.Body.Bytes(), []byte("8081")) {
		t.Fatal("internal endpoint identity leaked in public limits API")
	}
}
