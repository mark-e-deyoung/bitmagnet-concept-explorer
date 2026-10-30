package main

import (
	"net/http"
	"time"
)

type SweepLimits struct {
	Version                          string `json:"version"`
	MaxBitmagnetQueriesPerSweep      int    `json:"max_bitmagnet_queries_per_sweep"`
	MaxSearxngQueriesPerSweep        int    `json:"max_searxng_queries_per_sweep"`
	MaxConcurrentSweeps              int    `json:"max_concurrent_sweeps"`
	SweepDeadlineSeconds             int    `json:"sweep_deadline_seconds"`
	ClientTimeoutSeconds             int    `json:"client_timeout_seconds"`
	MaxSearxngJSONRequestsPerHour    int    `json:"max_searxng_json_requests_per_hour"`
	LocalSearxngJSONRequestsInWindow int    `json:"local_searxng_json_requests_in_window"`
	RemainingSearxngJSONRequests     int    `json:"remaining_searxng_json_requests"`
	SearxngCooldownSeconds           int64  `json:"searxng_cooldown_seconds"`
	SearxngSweepAdmitted             bool   `json:"searxng_sweep_admitted"`
	EgressPolicy                     string `json:"egress_policy"`
	Coverage                         string `json:"coverage"`
}

func sweepLimits() SweepLimits {
	v := SweepLimits{
		Version: "sweep-limits-v0", MaxBitmagnetQueriesPerSweep: 12,
		MaxSearxngQueriesPerSweep: 3, MaxSearxngJSONRequestsPerHour: 3,
		MaxConcurrentSweeps: maxConcurrentSweeps, SweepDeadlineSeconds: int(sweepExecutionDeadline / time.Second),
		ClientTimeoutSeconds:         int(sweepClientTimeout / time.Second),
		RemainingSearxngJSONRequests: 3, SearxngSweepAdmitted: privateSearxngSweepAllowed(),
		EgressPolicy: "one-shot opt-in; operator-attested private SearXNG; provider refusal stops source; no automatic retry or CAPTCHA bypass",
		Coverage:     "limits are local to this process only; not an instance-wide or upstream-provider guarantee; no recurring sweeps without shared admission",
	}
	ep, e := searXNGEndpoint()
	if e != nil {
		return v
	}
	searXNGPace.Lock()
	defer searXNGPace.Unlock()
	now := time.Now()
	for _, t := range searXNGPace.Recent[ep] {
		if now.Sub(t) < time.Hour {
			v.LocalSearxngJSONRequestsInWindow++
		}
	}
	v.RemainingSearxngJSONRequests = 3 - v.LocalSearxngJSONRequestsInWindow
	if v.RemainingSearxngJSONRequests < 0 {
		v.RemainingSearxngJSONRequests = 0
	}
	if d := time.Until(searXNGPace.BlockedUntil[ep]); d > 0 {
		v.SearxngCooldownSeconds = int64(d.Seconds()) + 1
	}
	return v
}
func registerSweepLimits(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sweeps/limits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, sweepLimits())
	})
}
