// Package cmd wires command-line flags to the UI and runs it.
package cmd

import (
	"flag"
	"runtime/debug"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/UltimateForm/protonshell/internal/servers"
	"github.com/UltimateForm/protonshell/internal/ui"
)

type options struct {
	list  string
	iface string
}

// Launch parses args (without the program name) and runs the UI until the
// user quits. linkedVersion is the version injected at link time, if any.
func Launch(args []string, linkedVersion string) error {
	opts := parseFlags(args)
	info, _ := debug.ReadBuildInfo()
	p := tea.NewProgram(ui.New(opts.list, opts.iface, resolveVersion(linkedVersion, info)))
	_, err := p.Run()
	return err
}

// parseFlags exits the process on -h or an unknown flag, like flag.Parse.
func parseFlags(args []string) options {
	var opts options
	fs := flag.NewFlagSet("protonshell", flag.ExitOnError)
	fs.StringVar(&opts.list, "list", servers.DefaultPath(), "path to the Proton VPN server list cache")
	fs.StringVar(&opts.iface, "iface", "proton0", "VPN network interface to measure speed on")
	_ = fs.Parse(args) // ExitOnError: never returns an error
	return opts
}

// resolveVersion prefers the version injected at link time, then the module
// version Go records for `go install …@vX.Y.Z` builds and VCS checkouts, and
// falls back to "dev".
func resolveVersion(linked string, info *debug.BuildInfo) string {
	v := linked
	if v == "" && info != nil && info.Main.Version != "(devel)" {
		v = info.Main.Version
	}
	if v == "" {
		return "dev"
	}
	// GoReleaser's {{.Version}} drops the tag's leading "v"; Go's keeps it.
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}
