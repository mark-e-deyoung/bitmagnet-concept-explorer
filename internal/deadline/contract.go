// Package deadline holds the user-facing request budget shared by the server
// package and the standalone CLI.
package deadline

import "time"

// ClientTimeout gives the server time to return bounded full or partial sweep
// results after its execution deadline and HTTP response allowance.
// ClientTimeout includes the server's body-read and execution budgets plus
// the time needed to serialize and deliver the response.
const ClientTimeout = 80 * time.Second
