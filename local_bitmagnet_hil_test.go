package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// TestLocalBitmagnetVersionHIL is an operator-gated, non-mutating smoke for a
// repository-admitted TrueNAS-adjacent GARM runner. Ordinary go test skips it.
// It retrieves ONLY Bitmagnet's GraphQL version, never corpus rows or secrets.
func TestLocalBitmagnetVersionHIL(t *testing.T) {
	if os.Getenv("BCE_LOCAL_BITMAGNET_HIL") != "1" {
		t.Skip("local Bitmagnet HIL not explicitly enabled")
	}
	raw := os.Getenv("BCE_BITMAGNET_GRAPHQL")
	if raw == "" {
		t.Fatal("BCE_BITMAGNET_GRAPHQL is required for explicit HIL")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "/graphql" {
		t.Fatal("explicit HIL destination is not a valid trusted GraphQL route")
	}
	host := net.ParseIP(u.Hostname())
	if host == nil || !(host.IsPrivate() || host.IsLoopback()) {
		t.Fatal("explicit HIL destination must be a literal private or loopback IP")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	// Pin resolved destination for the only request: no HTTP proxy, no redirect
	// and no untrusted DNS resolution. The private IP is never printed to logs.
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	tr := &http.Transport{Proxy: nil, DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, net.JoinHostPort(host.String(), port))
		},
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Timeout: 5 * time.Second, Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBufferString(`{"query":"query LocalReadOnlyHIL { version }"}`))
	if err != nil {
		t.Fatal("could not build bounded GraphQL HIL request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("local GraphQL version probe failed or timed out; no corpus access was attempted")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("local Bitmagnet GraphQL status %d; no corpus query was attempted", resp.StatusCode)
	}
	var parsed struct {
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	d := json.NewDecoder(io.LimitReader(resp.Body, 8192))
	if e := d.Decode(&parsed); e != nil || len(parsed.Errors) > 0 || parsed.Data.Version == "" {
		t.Fatal("local GraphQL version contract invalid or returned errors")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		t.Fatal("local GraphQL version response contained trailing data")
	}
	t.Log("local Bitmagnet GraphQL read-only version probe PASS (response payload redacted)")
}
