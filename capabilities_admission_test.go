package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// An advertised execute capability must not bypass the exact source admission
// rules enforced again by the execution route. This test performs no network IO.
func TestCapabilitiesRespectBitmagnetEndpointAdmission(t *testing.T) {
	cases := []struct {
		name        string
		endpoint    string
		wantExecute bool
	}{
		{"unset", "", false},
		{"public_dns", "https://example.org/graphql", false},
		{"public_ip", "https://8.8.8.8/graphql", false},
		{"wrong_path", "http://127.0.0.1:8080/other", false},
		{"local_literal", "http://127.0.0.1:8080/graphql", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("BCE_BITMAGNET_GRAPHQL", tc.endpoint)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
			response := httptest.NewRecorder()
			handler().ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("capability status = %d", response.Code)
			}
			var payload struct {
				Execute          bool   `json:"execute"`
				TargetValidation string `json:"execution_target_validation"`
			}
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode capability JSON: %v", err)
			}
			if payload.Execute != tc.wantExecute {
				t.Fatalf("execute capability = %t, want %t", payload.Execute, tc.wantExecute)
			}
			if payload.TargetValidation != "configuration_only_no_live_probe" {
				t.Fatalf("unexpected target validation contract: %q", payload.TargetValidation)
			}
		})
	}
}
