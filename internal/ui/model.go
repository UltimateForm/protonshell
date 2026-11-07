// Package ui is the Bubble Tea front end.
package ui

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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
	return tea.Batch(tick(), loadList(m.listPath, time.Time{}), fetchStatus())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampCursor()
		return m, nil

	case tickMsg:
		m.ticks++
		m.rx, m.tx, m.speedOK = m.sampler.Sample()
		cmds := []tea.Cmd{tick()}
		if !m.listInFlight {
			m.listInFlight = true
			cmds = append(cmds, loadList(m.listPath, m.listMod))
		}
		if m.ticks%statusEvery == 0 && !m.statusInFlight && m.busy == "" {
			m.statusInFlight = true
			cmds = append(cmds, fetchStatus())
		}
		return m, tea.Batch(cmds...)

	case listMsg:
		m.listInFlight = false
		if msg.err != nil {
			m.listErr = "server list: " + msg.err.Error()
			return m, nil
		}
		m.listErr = ""
		if msg.list != nil {
			m.all = msg.list.Available()
			m.listMod = msg.mod
			m.rebuild()
		}
		return m, nil

	case statusMsg:
		m.statusInFlight = false
		if msg.err != nil {
			m.setError("status: " + msg.err.Error())
			return m, nil
		}
		m.status, m.statusKnown = msg.status, true
		return m, nil

	case actionMsg:
		m.busy = ""
		if msg.err != nil {
			m.setError(msg.err.Error())
		} else {
			m.message, m.isError = firstLine(msg.out), false
		}
		m.statusInFlight = true
		return m, fetchStatus()

	case tea.KeyMsg:
		if m.filter.Focused() {
			return m.updateFilter(msg)
		}
		return m.updateKeys(msg)
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.filter.SetValue("")
		m.filter.Blur()
		m.rebuild()
		return m, nil
	case "enter":
		m.filter.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.rebuild()
	return m, cmd
}

func (m Model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	page := m.visibleRows()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "pgup":
		m.cursor -= page
	case "pgdown":
		m.cursor += page
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.rows) - 1
	case "/":
		return m, m.filter.Focus()
	case "esc":
		m.filter.SetValue("")
		m.rebuild()
	case "s":
		m.sort = (m.sort + 1) % sortModes
		m.rebuild()
	case "r":
		if !m.statusInFlight {
			m.statusInFlight = true
			return m, fetchStatus()
		}
	case "enter":
		if m.busy != "" || len(m.rows) == 0 {
			return m, nil
		}
		name := m.rows[m.cursor].Name
		m.busy = "Connecting to " + name + "…"
		return m, connect(name)
	case "d":
		if m.busy != "" {
			return m, nil
		}
		m.busy = "Disconnecting…"
		return m, disconnect()
	}
	m.clampCursor()
	return m, nil
}

// rebuild reapplies the filter and sort, keeping the cursor on the same
// server when it is still in the result.
func (m *Model) rebuild() {
	var selected string
	if m.cursor < len(m.rows) {
		selected = m.rows[m.cursor].Name
	}

	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.rows = m.rows[:0]
	for _, s := range m.all {
		if query == "" || matches(s, query) {
			m.rows = append(m.rows, s)
		}
	}
	slices.SortStableFunc(m.rows, m.sort.compare)

	m.cursor = 0
	if selected != "" {
		if i := slices.IndexFunc(m.rows, func(s servers.Logical) bool { return s.Name == selected }); i >= 0 {
			m.cursor = i
		}
	}
	m.clampCursor()
}

func matches(s servers.Logical, query string) bool {
	return strings.Contains(strings.ToLower(s.Name), query) ||
		strings.Contains(strings.ToLower(s.City), query) ||
		strings.ToLower(s.ExitCountry) == query
}

func (s sortMode) compare(a, b servers.Logical) int {
	switch s {
	case sortLoad:
		return cmp.Or(cmp.Compare(a.Load, b.Load), compareNames(a.Name, b.Name))
	case sortName:
		return compareNames(a.Name, b.Name)
	case sortCountry:
		return cmp.Or(cmp.Compare(a.ExitCountry, b.ExitCountry), compareNames(a.Name, b.Name))
	default:
		return cmp.Or(cmp.Compare(a.Score, b.Score), compareNames(a.Name, b.Name))
	}
}

// compareNames orders "PT#7" before "PT#10".
func compareNames(a, b string) int {
	aPrefix, aNum, _ := strings.Cut(a, "#")
	bPrefix, bNum, _ := strings.Cut(b, "#")
	if c := cmp.Compare(aPrefix, bPrefix); c != 0 {
		return c
	}
	an, aErr := strconv.Atoi(aNum)
	bn, bErr := strconv.Atoi(bNum)
	if aErr != nil || bErr != nil {
		return cmp.Compare(aNum, bNum)
	}
	return cmp.Compare(an, bn)
}

func (m *Model) clampCursor() {
	m.cursor = max(0, min(m.cursor, len(m.rows)-1))
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	m.offset = max(0, min(m.offset, len(m.rows)-visible))
}

func (m Model) visibleRows() int {
	return max(1, m.height-chromeLines)
}

func (m *Model) setError(text string) {
	m.message, m.isError = text, true
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}
