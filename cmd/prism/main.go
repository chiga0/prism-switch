package main

import "github.com/chiga0/prism-switch/internal/cli"

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersion(version, commit, date)
	cli.Execute()
}
