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
func main() {
	lockFile := flag.String("lock", "", "immutable lock JSON")
	packagesFile := flag.String("packages", "", "offline package array JSON")
	roots := flag.String("root", "", "root package ID")
	public := flag.Bool("public", true, "require public dependency closure")
	format := flag.String("format", "readmodel", "readmodel or bce-preview (public research-only)")
	flag.Parse()
	if *lockFile == "" || *packagesFile == "" || *roots == "" {
		fmt.Fprintln(os.Stderr, "required: -lock -packages -root")
		os.Exit(2)
	}
	var l cpepack.Lock
	var ps []cpepack.Package
	if err := read(*lockFile, &l); err != nil {
		fmt.Fprintln(os.Stderr, "invalid lock:", err)
		os.Exit(2)
	}
	if err := read(*packagesFile, &ps); err != nil {
		fmt.Fprintln(os.Stderr, "invalid packages:", err)
		os.Exit(2)
	}
	projection, err := cpepack.Build(ps, l, []string{*roots}, *public)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *format == "bce-preview" {
		if !*public {
			fmt.Fprintln(os.Stderr, "bce-preview must be public")
			os.Exit(2)
		}
		raw, _, err := cpepack.BuildBCEPreview(ps, l, []string{*roots})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if _, err = os.Stdout.Write(append(raw, '\n')); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *format != "readmodel" {
		fmt.Fprintln(os.Stderr, "unsupported format")
		os.Exit(2)
	}
	out := json.NewEncoder(os.Stdout)
	out.SetIndent("", "  ")
	if err := out.Encode(projection); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
