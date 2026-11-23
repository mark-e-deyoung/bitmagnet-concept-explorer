// cpepack compiles a pinned offline research-only read model; no network access.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/cpepack"
)

func read(path string, out any) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if e != nil || len(raw) > 2<<20 {
		return fmt.Errorf("input too large or unreadable")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if e := dec.Decode(out); e != nil {
		return e
	}
	var trailing any
	if e := dec.Decode(&trailing); e != io.EOF {
		return fmt.Errorf("trailing or invalid input: %v", e)
	}
	return nil
}
// run implements the existing offline package build and the opt-in research-only
// subset/representation view using the SAME cpepack BuildSubset/BuildPinnedSubset
// library as direct callers. This command does not issue accepted releases.
func run(args []string, stdout, stderr io.Writer) int {
    flags := flag.NewFlagSet("cpepack", flag.ContinueOnError)
    flags.SetOutput(stderr)
    lockFile := flags.String("lock", "", "immutable research lock JSON")
    packagesFile := flags.String("packages", "", "offline research package array JSON")
    roots := flags.String("root", "", "root package ID")
    public := flags.Bool("public", true, "require public dependency closure")
    selectorFile := flags.String("selector", "", "optional research-only subset selector JSON")
    refsFile := flags.String("representations", "", "optional research-only representation reference array JSON")
    expectedSource := flags.String("expected-source-sha256", "", "independently admitted source-lock SHA-256 (hex)")
    expectedSelection := flags.String("expected-selection-sha256", "", "independently admitted selection SHA-256 (hex)")
    if err := flags.Parse(args); err != nil {
        return 2
    }
    if len(flags.Args()) != 0 || *lockFile == "" || *packagesFile == "" || *roots == "" {
        fmt.Fprintln(stderr, "required: -lock -packages -root; unexpected positional arguments refused")
        return 2
    }
    if (*selectorFile == "") != (*refsFile == "") {
        fmt.Fprintln(stderr, "selector and representations must both be supplied")
        return 2
    }
    if (*expectedSource == "") != (*expectedSelection == "") {
        fmt.Fprintln(stderr, "both independent expected digests must be supplied together")
        return 2
    }
    if *selectorFile == "" && (*expectedSource != "" || *expectedSelection != "") {
        fmt.Fprintln(stderr, "selection pins require selector and representations")
        return 2
    }
    var l cpepack.Lock
    var ps []cpepack.Package
    if err := read(*lockFile, &l); err != nil {
        fmt.Fprintln(stderr, "invalid lock:", err)
        return 2
    }
    if err := read(*packagesFile, &ps); err != nil {
        fmt.Fprintln(stderr, "invalid packages:", err)
        return 2
    }

    out := json.NewEncoder(stdout)
    out.SetIndent("", "  ")
    if *selectorFile == "" {
        projection, err := cpepack.Build(ps, l, []string{*roots}, *public)
        if err != nil {
            fmt.Fprintln(stderr, err)
            return 1
        }
        if err := out.Encode(projection); err != nil {
            fmt.Fprintln(stderr, err)
            return 1
        }
        return 0
    }

    var selector cpepack.SubsetSelector
    var refs []cpepack.RepresentationRef
    if err := read(*selectorFile, &selector); err != nil {
        fmt.Fprintln(stderr, "invalid selector:", err)
        return 2
    }
    if err := read(*refsFile, &refs); err != nil {
        fmt.Fprintln(stderr, "invalid representation references:", err)
        return 2
    }
    // Unpinned mode is strictly research-only exploration, never an
    // approved release. Pinned mode compares against independently supplied
    // digests; it does not establish whose authority provided those values.
    var view cpepack.SubsetView
    var err error
    if *expectedSource != "" {
        view, err = cpepack.BuildPinnedSubset(ps, l, []string{*roots},
            selector, refs, *public, *expectedSource, *expectedSelection)
    } else {
        view, err = cpepack.BuildSubset(ps, l, []string{*roots}, selector, refs, *public)
    }
    if err != nil {
        fmt.Fprintln(stderr, err)
        return 1
    }
    if err := out.Encode(view); err != nil {
        fmt.Fprintln(stderr, err)
        return 1
    }
    return 0
}

func main() {
    os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
