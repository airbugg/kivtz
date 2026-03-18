package main

import (
	"os"

	"github.com/airbugg/kivtz/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersion(version, commit, date)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
