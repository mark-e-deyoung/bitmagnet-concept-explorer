package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateServiceOriginAllowsLoopbackPortsAndRejectsUnsafeHTTP(t *testing.T) {
	accepted := []string{
		"http://127.0.0.1:3086",
		"http://127.0.0.1:3087",
		"http://localhost:9000",
		"http://[::1]:43123",
		"https://example.invalid:443",
	}
	for _, raw := range accepted {
		if _, err := validateServiceOrigin(raw); err != nil {
			t.Errorf("validateServiceOrigin(%q) error = %v", raw, err)
		}
	}
	rejected := []string{
		"http://example.invalid:3086",
		"http://user:password@127.0.0.1:3087",
		"http://127.0.0.1:3087/path",
		"http://127.0.0.1:3087/?query=1",
		"http://127.0.0.1:3087/#fragment",
		"ftp://127.0.0.1:3087",
	}
	for _, raw := range rejected {
		if _, err := validateServiceOrigin(raw); err == nil {
			t.Errorf("validateServiceOrigin(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestCLIJSONInputLimitsMatchRouteContracts(t *testing.T) {
	operations := map[string]int{
		"observed-graph":    1 << 20,
		"cpe-preflight":     32 << 10,
		"infer":             8 << 10,
		"sweep-plan":        16 << 10,
		"sweep":             16 << 10,
		"discovery-preview": 16 << 10,
	}
	for operation, want := range operations {
		limit, ok := operationInputLimit(operation)
		if !ok || limit != int64(want) {
			t.Fatalf("operationInputLimit(%q)=(%d,%v), want (%d,true)", operation, limit, ok, want)
		}
		if _, err := readOperationInput(strings.NewReader(strings.Repeat("x", want)), operation); err != nil {
			t.Fatalf("input exactly at %d bytes rejected for %s: %v", want, operation, err)
		}
		if _, err := readOperationInput(strings.NewReader(strings.Repeat("x", want+1)), operation); err == nil {
			t.Fatalf("input over %d bytes accepted for %s", want, operation)
		}
	}
	if _, err := operationInputLimit("query-plan"); err {
		t.Fatal("unexpected JSON input contract for flag-built query plan")
	}
	if _, err := readOperationInput(strings.NewReader(""), "sweep"); err == nil {
		t.Fatal("empty JSON input accepted")
	}
}

func TestCLIRejectsOversizedHTTP200ResponseWithoutReturningPrefix(t *testing.T) {
	responseBody := append([]byte(`{"state":"complete"}`), bytes.Repeat([]byte(" "), maxCLIResponseBytes)...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("synthetic server returned %d", resp.StatusCode)
	}
	output, err := readServiceResponse(resp.Body)
	if err == nil || len(output) != 0 {
		t.Fatalf("oversized 200 response returned a truncated prefix (%d bytes), err=%v", len(output), err)
	}

	valid := []byte(`{"state":"complete"}`)
	output, err = readServiceResponse(strings.NewReader(string(valid)))
	if err != nil || !json.Valid(output) {
		t.Fatalf("small valid JSON response rejected: bytes=%q err=%v", output, err)
	}
}
