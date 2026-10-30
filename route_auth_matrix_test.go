package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthenticationCoversRegisteredRouteMatrix(t *testing.T) {
	t.Setenv("BCE_BITMAGNET_GRAPHQL", "http://127.0.0.1:1/graphql")
	const user, password = "matrix-user", "matrix-password-123456"
	app := authenticated(handler(), user, password)
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/"},
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/api/v1/capabilities"},
		{http.MethodGet, "/api/v1/concepts"},
		{http.MethodGet, "/api/v1/concepts/missing"},
		{http.MethodGet, "/api/v1/sweeps/limits"},
		{http.MethodPost, "/api/v1/query/plan"},
		{http.MethodPost, "/api/v1/query/execute"},
		{http.MethodPost, "/api/v1/query/facets"},
		{http.MethodPost, "/api/v1/query/observed-graph"},
		{http.MethodPost, "/api/v1/cpe/adapter-plan/preflight"},
		{http.MethodPost, "/api/v1/cpe/adapter-plan/lower-preview"},
		{http.MethodPost, "/api/v1/discovery-proposals/preview"},
		{http.MethodPost, "/api/v1/inference/propose"},
		{http.MethodPost, "/api/v1/sweeps/plan"},
		{http.MethodPost, "/api/v1/sweeps/execute"},
		{http.MethodPost, "/api/v1/sweeps/discovery/preview"},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			unauth := httptest.NewRecorder()
			app.ServeHTTP(unauth, httptest.NewRequest(route.method, route.path, nil))
			if route.method == http.MethodGet && route.path == "/healthz" {
				if unauth.Code != http.StatusOK {
					t.Fatalf("healthz status=%d, want 200", unauth.Code)
				}
				return
			}
			if unauth.Code != http.StatusUnauthorized || unauth.Header().Get("WWW-Authenticate") == "" || unauth.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unauthenticated status=%d challenge=%q cache=%q", unauth.Code, unauth.Header().Get("WWW-Authenticate"), unauth.Header().Get("Cache-Control"))
			}
			if route.method != http.MethodPost {
				return
			}
			req := httptest.NewRequest(route.method, route.path, strings.NewReader("{"))
			req.SetBasicAuth(user, password)
			authenticatedResponse := httptest.NewRecorder()
			app.ServeHTTP(authenticatedResponse, req)
			if authenticatedResponse.Code != http.StatusBadRequest {
				t.Fatalf("authenticated malformed request status=%d body=%s, want 400", authenticatedResponse.Code, authenticatedResponse.Body.String())
			}
		})
	}
}
