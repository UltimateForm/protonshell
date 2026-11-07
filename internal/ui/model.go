// Package ui is the Bubble Tea front end.
package ui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/UltimateForm/protonshell/internal/netstat"
	"github.com/UltimateForm/protonshell/internal/servers"
	"github.com/UltimateForm/protonshell/internal/vpn"
)

// statusEvery is how many ticks pass between `protonvpn status` calls; the
// CLI is too slow to start to run it every second.
const statusEvery = 5

// chromeLines is the number of screen lines not used by server rows.
const chromeLines = 5

type sortMode int

const (
	sortScore sortMode = iota
	sortLoad
	sortName
	sortCountry
	sortModes
)

func (s sortMode) String() string {
	return [...]string{"score", "load", "name", "country"}[s]
}

type Model struct {
	listPath string
	iface    string

	all  []servers.Logical
	rows []servers.Logical // all, filtered and sorted

	cursor, offset int
	width, height  int

	filter textinput.Model
	sort   sortMode

	status         vpn.Status
	statusKnown    bool
	statusInFlight bool

	sampler *netstat.Sampler
	rx, tx  float64
	speedOK bool

	listMod      time.Time
	listInFlight bool

	// listErr is kept apart from message: the CLI rewrites the list in place,
	// so a failed read is usually transient and clears on the next tick.
	listErr string

	busy    string // action in progress, e.g. "Connecting to ES#220…"
	message string
	isError bool
	ticks   int
}

func New(listPath, iface string) Model {
	filter := textinput.New()
	filter.Prompt = "/"
	filter.Placeholder = "name, country code or city"
	return Model{
		listPath: listPath,
		iface:    iface,
		filter:   filter,
		sampler:  netstat.NewSampler(iface),
	}
}

func (m Model) Init() tea.Cmd {
	return initCmds(m)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return update(m, msg)
}

func (m Model) View() tea.View {
	v := tea.NewView(render(m))
	v.AltScreen = true
	return v
}
