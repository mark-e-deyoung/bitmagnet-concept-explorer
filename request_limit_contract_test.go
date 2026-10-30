package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
)

func TestRegisteredPostRoutesEnforceSharedBodyBudgets(t *testing.T) {
	t.Setenv("BCE_BITMAGNET_GRAPHQL", "http://127.0.0.1:1/graphql")
	testCases := []struct {
		path  string
		limit int64
	}{
		{"POST /api/v1/query/plan", contracts.QueryJSONBytes},
		{"POST /api/v1/query/execute", contracts.QueryJSONBytes},
		{"POST /api/v1/query/facets", contracts.FacetJSONBytes},
		{"POST /api/v1/query/observed-graph", contracts.ObservedGraphJSONBytes},
		{"POST /api/v1/cpe/adapter-plan/preflight", contracts.CPEPreflightJSONBytes},
		{"POST /api/v1/cpe/adapter-plan/lower-preview", contracts.CPELowerPreviewJSONBytes},
		{"POST /api/v1/discovery-proposals/preview", contracts.LegacyDiscoveryJSONBytes},
		{"POST /api/v1/inference/propose", contracts.InferenceJSONBytes},
		{"POST /api/v1/sweeps/plan", contracts.SweepJSONBytes},
		{"POST /api/v1/sweeps/execute", contracts.SweepJSONBytes},
		{"POST /api/v1/sweeps/discovery/preview", contracts.SweepDiscoveryJSONBytes},
	}
	app := handler()
	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			body := strings.Repeat(" ", int(tc.limit+1))
			req := httptest.NewRequest(http.MethodPost, strings.TrimPrefix(tc.path, "POST "), strings.NewReader(body))
			response := httptest.NewRecorder()
			app.ServeHTTP(response, req)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("oversized request status=%d limit=%d body=%s", response.Code, tc.limit, response.Body.String())
			}
		})
	}
}
