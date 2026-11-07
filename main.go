package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/UltimateForm/protonshell/internal/servers"
	"github.com/UltimateForm/protonshell/internal/ui"
)

func main() {
	list := flag.String("list", servers.DefaultPath(), "path to the Proton VPN server list cache")
	iface := flag.String("iface", "proton0", "VPN network interface to measure speed on")
	flag.Parse()

	p := tea.NewProgram(ui.New(*list, *iface), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
