package main

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlan(t *testing.T) {
	p, err := makePlan(Request{Query: "  alpha ", Expansions: []Expansion{{Term: "beta", Language: "de", Provenance: "fixture"}, {Term: "ALPHA", Provenance: "fixture"}}})
	if err != nil || !p.ReadOnly || len(p.Tracks) != 2 || len(p.Tracks[1].Terms) != 1 || p.Tracks[1].Terms[0] != "beta" {
		t.Fatalf("unexpected plan: %#v %v", p, err)
	}
}
func TestInvalidExpansion(t *testing.T) {
	if _, err := makePlan(Request{Query: "a", Expansions: []Expansion{{Term: "b"}}}); err == nil {
		t.Fatal("missing provenance accepted")
	}
}
func TestCapabilitiesAndPlan(t *testing.T) {
	s := httptest.NewServer(handler())
	defer s.Close()
	r, err := http.Get(s.URL + "/api/v1/capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var c map[string]any
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c["execute"] != false || c["mutation"] != false {
		t.Fatalf("bad capabilities %v %v", c, err)
	}
	r2, err := http.Post(s.URL+"/api/v1/query/plan", "application/json", bytes.NewBufferString(`{"query":"sample"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != 200 {
		t.Fatalf("unexpected %d", r2.StatusCode)
	}
}

func TestListenRequiresCredentialsOffLoopback(t *testing.T) {
	if err := validateListenAuth("0.0.0.0:3086", "", ""); err == nil {
		t.Fatal("non-loopback bind accepted without credentials")
	}
	if err := validateListenAuth("127.0.0.1:3086", "", ""); err != nil {
		t.Fatalf("loopback bind rejected without credentials: %v", err)
	}
	if err := validateListenAuth("0.0.0.0:3086", "explorer", "0123456789abcdef"); err != nil {
		t.Fatalf("authenticated non-loopback bind rejected: %v", err)
	}
	for _, tc := range []struct{ user, password string }{{"explorer", "short"}, {"user:name", "0123456789abcdef"}, {"", "0123456789abcdef"}, {"explorer", ""}} {
		if err := validateListenAuth("127.0.0.1:3086", tc.user, tc.password); err == nil {
			t.Errorf("invalid credential pair accepted: %#v", tc)
		}
	}
}

func TestAuthenticationProtectsAllButHealth(t *testing.T) {
	const user, password = "explorer", "0123456789abcdef"
	h := authenticated(handler(), user, password)
	for _, tc := range []struct {
		path       string
		user       string
		password   string
		statusCode int
	}{{"/api/v1/capabilities", "", "", http.StatusUnauthorized}, {"/api/v1/capabilities", user, "wrong-password-000", http.StatusUnauthorized}, {"/api/v1/capabilities", user, password, http.StatusOK}, {"/healthz", "", "", http.StatusOK}} {
		r := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if tc.user != "" {
			r.SetBasicAuth(tc.user, tc.password)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.statusCode {
			t.Errorf("%s auth status=%d, want %d", tc.path, w.Code, tc.statusCode)
		}
	}
}

func TestContainerHealthcheck(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler()}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()
	if err := checkHealth(listener.Addr().String()); err != nil {
		t.Fatalf("healthcheck failed against healthy server: %v", err)
	}
	if err := checkHealth("invalid-address"); err == nil {
		t.Fatal("healthcheck accepted invalid address")
	}
}
