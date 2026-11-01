// Package contracts defines request budgets shared by HTTP routes and clients.
package contracts

const (
	QueryJSONBytes           int64 = 16 << 10
	FacetJSONBytes           int64 = 8 << 10
	ObservedGraphJSONBytes   int64 = 1 << 20
	CPEPreflightJSONBytes    int64 = 32 << 10
	CPELowerPreviewJSONBytes int64 = 32 << 10
	InferenceJSONBytes       int64 = 8 << 10
	LegacyDiscoveryJSONBytes int64 = 8 << 10
	SweepJSONBytes           int64 = 16 << 10
	SweepDiscoveryJSONBytes  int64 = 16 << 10
)

// CLIInputLimit returns the exact body budget for CLI operations that read JSON.
func CLIInputLimit(operation string) (int64, bool) {
	switch operation {
	case "observed-graph":
		return ObservedGraphJSONBytes, true
	case "cpe-preflight":
		return CPEPreflightJSONBytes, true
	case "infer":
		return InferenceJSONBytes, true
	case "sweep-plan", "sweep":
		return SweepJSONBytes, true
	case "discovery-preview":
		return SweepDiscoveryJSONBytes, true
	default:
		return 0, false
	}
}
