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

// TestPublicCloudEphemeralDeployment runs the compiled service and CLI in
// separate OS processes with a loopback-only synthetic Bitmagnet backend.
// No real corpus, public ingress, credential, mutation or release is involved.
func TestPublicCloudEphemeralDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-process cloud rep requires an ordinary test run")
	}
	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			http.Error(w, "unsupported synthetic route", http.StatusNotFound)
			return
		}
		var body struct {
			Query     string `json:"query"`
			Variables struct {
				Input struct {
					QueryString string `json:"queryString"`
				} `json:"input"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&body); err != nil || !strings.HasPrefix(body.Query, "query Search(") {
			http.Error(w, "non-read-only or malformed synthetic GraphQL request", http.StatusBadRequest)
			return
		}
		upstreamRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if body.Variables.Input.QueryString == "upstream-failure" {
			_, _ = io.WriteString(w, `{"errors":[{"message":"synthetic failure"}]}`)
			return
		}
		var items []map[string]any
		switch body.Variables.Input.QueryString {
		case "Barsoom":
			items = []map[string]any{
				{"title": "Synthetic Barsoom record", "contentType": "OTHER", "torrent": map[string]any{"infoHash": "aaa111", "name": "synthetic-only"}},
				{"title": "Synthetic shared record", "contentType": "OTHER", "torrent": map[string]any{"infoHash": "bbb222", "name": "synthetic-only"}},
			}
		case "Mars":
			items = []map[string]any{
				{"title": "Synthetic shared record", "contentType": "OTHER", "torrent": map[string]any{"infoHash": "bbb222", "name": "synthetic-only"}},
				{"title": "Synthetic Mars record", "contentType": "OTHER", "torrent": map[string]any{"infoHash": "ccc333", "name": "synthetic-only"}},
			}
		default:
			items = []map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"torrentContent": map[string]any{"search": map[string]any{"hasNextPage": false, "items": items}}}})
	}))
	defer upstream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	tmp := t.TempDir()
	serviceBinary, cliBinary := filepath.Join(tmp, "bce-service"), filepath.Join(tmp, "bce-cli")
	for _, build := range []struct{ target, pkg string }{{serviceBinary, "."}, {cliBinary, "./cmd/bce"}} {
		cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", build.target, build.pkg)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "GOFLAGS=-mod=readonly")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("offline service/CLI build failed: %v: %s", err, output)
		}
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close() // The actual service binary binds this randomly allocated port.

	const user, password = "cloud-rep", "synthetic-public-only-credential"
	service := exec.CommandContext(ctx, serviceBinary, "-listen", addr)
	service.Env = append(os.Environ(), "BCE_BITMAGNET_GRAPHQL="+upstream.URL+"/graphql", "BCE_AUTH_USER="+user, "BCE_AUTH_PASSWORD="+password, "BCE_SEARXNG_URL=", "BCE_CPE_GRAPH_FILE=", "BCE_CPE_GRAPH_SHA256=")
	var serverLog bytes.Buffer
	service.Stderr = &serverLog
	if err := service.Start(); err != nil {
		t.Fatalf("start compiled service: %v", err)
	}
	defer func() {
		if service.Process != nil {
			_ = service.Process.Kill()
			_ = service.Wait()
		}
	}()
	origin := "http://" + addr
	client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}
	var ready bool
	for i := 0; i < 80; i++ {
		resp, err := client.Get(origin + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal("service startup exceeded deadline")
		case <-time.After(50 * time.Millisecond):
		}
	}
	if !ready {
		t.Fatalf("compiled service did not become healthy: %s", serverLog.String())
	}

	request := func(method, path, payload string, authorized bool) (int, []byte) {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, origin+path, strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		if authorized {
			req.SetBasicAuth(user, password)
		}
		if payload != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, body
	}
	if code, _ := request("GET", "/api/v1/capabilities", "", false); code != 401 {
		t.Fatalf("unauthenticated capability status=%d, want 401", code)
	}
	if code, body := request("GET", "/api/v1/capabilities", "", true); code != 200 || !bytes.Contains(body, []byte(`"mutation":false`)) || !bytes.Contains(body, []byte(`"execute":true`)) {
		t.Fatalf("capability/source-boundary check failed: HTTP %d: %s", code, body)
	}
	if code, body := request("GET", "/", "", true); code != 200 || !bytes.Contains(bytes.ToLower(body), []byte("concept explorer")) {
		t.Fatalf("real browser document unavailable: HTTP %d", code)
	}
	const dualTrack = `{"query":"Barsoom","expansions":[{"term":"Mars","provenance":"manual-unverified"}]}`
	code, body := request("POST", "/api/v1/query/execute", dualTrack, true)
	var result Execution
	if code != 200 || json.Unmarshal(body, &result) != nil {
		t.Fatalf("real execute endpoint failed: HTTP %d: %s", code, body)
	}
	if result.State != "complete" || !result.ReadOnly || len(result.Results) != 3 || result.TrackCounts["direct"] != 2 || result.TrackCounts["concept"] != 2 || len(result.Coverage) != 2 {
		t.Fatalf("unexpected two-track synthetic retrieval: %+v", result)
	}
	var shared bool
	for _, observation := range result.Results {
		if observation.InfoHash == "bbb222" && len(observation.Tracks) == 2 {
			shared = true
		}
	}
	if !shared {
		t.Fatal("shared infohash was not deduplicated across both tracks")
	}

	cli := exec.CommandContext(ctx, cliBinary, "-server", origin, "-operation", "execute", "-query", "Barsoom", "-expand", "Mars")
	cli.Env = append(os.Environ(), "BCE_AUTH_USER="+user, "BCE_AUTH_PASSWORD="+password)
	cliOutput, err := cli.CombinedOutput()
	if err != nil {
		t.Fatalf("real CLI query failed: %v: %s", err, cliOutput)
	}
	var cliResult Execution
	if err := json.Unmarshal(bytes.TrimSpace(cliOutput), &cliResult); err != nil || cliResult.State != "complete" || len(cliResult.Results) != 3 {
		t.Fatalf("CLI/API disagreement: %v: %s", err, cliOutput)
	}

	code, body = request("POST", "/api/v1/query/execute", `{"query":"upstream-failure"}`, true)
	var failed Execution
	if code != 200 || json.Unmarshal(body, &failed) != nil || failed.State != "failed" || len(failed.Results) != 0 || len(failed.Failures) != 1 {
		t.Fatalf("upstream error was masked as no-hit: HTTP %d: %s", code, body)
	}
	if upstreamRequests.Load() < 5 {
		t.Fatalf("expected real GraphQL requests from service/CLI, observed %d", upstreamRequests.Load())
	}
	t.Logf("PUBLIC-CLOUD EPHEMERAL LOOPBACK REP PASS: compiled service+CLI, authenticated HTTP/browser, real two-track GraphQL, overlapping hashes, upstream error preserved; synthetic requests=%d; no external network, writes, release or persistent deployment", upstreamRequests.Load())
}
