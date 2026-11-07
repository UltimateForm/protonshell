package main

import (
	"fmt"
	"os"

	"github.com/UltimateForm/protonshell/cmd"
)

// version is set at release time by GoReleaser's default
// https://goreleaser.com/customization/builds/builders/go/#options
// -ldflags "-X main.version={{.Version}}"; it is empty in other builds.
var version string

func main() {
	if err := cmd.Launch(os.Args[1:], version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
