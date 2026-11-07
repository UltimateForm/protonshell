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
	sortModes
)

func (s sortMode) String() string {
	return [...]string{"cost", "load", "name"}[s] // wow pretty smart
}

type countrySortMode int

const (
	countrySortBest countrySortMode = iota
	countrySortName
	countrySortServers
	countrySortLoad
	countrySortModes
)

func (s countrySortMode) String() string {
	return [...]string{"cost", "name", "servers", "load"}[s]
}

// pane is the table that has keyboard focus.
type pane int

const (
	paneCountries pane = iota
	paneServers
)

// country is one row of the country pane: the matched servers that share an
// ExitCountry.
type country struct {
	Code    string
	Servers int
	Best    float64 // lowest score; lower is better
	AvgLoad int
}

type Model struct {
	listPath string
	iface    string

	all       []servers.Logical
	matched   []servers.Logical // all, filtered
	countries []country         // matched, grouped by ExitCountry and sorted
	rows      []servers.Logical // matched in the selected country, sorted

	focus                        pane
	countryCursor, countryOffset int
	cursor, offset               int // server pane
	width, height                int

	filter      textinput.Model
	sort        sortMode
	countrySort countrySortMode

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
