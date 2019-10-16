package main

// https://github.com/golang/go/issues/34053#issuecomment-532386030

import (
	"flag"

	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(
		shadow.Analyzer,
		// ... other custom analyzers
	)
}

func init() {
	// go vet always adds this flag for certain packages in the standard library,
	// which causes "flag provided but not defined" errors when running with
	// custom vet tools.
	// So we just declare it here and swallow the flag.
	// See https://github.com/golang/go/issues/34053 for details.
	// TODO: Remove this once above issue is resolved.
	flag.String("unsafeptr", "", "")
}
