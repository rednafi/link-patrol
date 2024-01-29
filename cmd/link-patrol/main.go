package main

import (
	"os"
	"text/tabwriter"

	"github.com/rednafi/link-patrol/src"
)

// Ldflags filled by goreleaser
var version string = "sentinel"

func main() {
	w := tabwriter.NewWriter(
		os.Stdout,
		0,
		4,
		4,
		' ',
		0,
	)
	defer w.Flush()
	src.CLI(
		w,
		version,
		os.Exit,
	)
}
