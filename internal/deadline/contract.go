// Package deadline holds the user-facing request budget shared by the server
// package and the standalone CLI.
package deadline

import "time"

// ClientTimeout gives the server time to return bounded full or partial sweep
// results after its execution deadline and HTTP response allowance.
const ClientTimeout = 65 * time.Second
