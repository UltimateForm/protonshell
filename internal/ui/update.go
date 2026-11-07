package ui

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/UltimateForm/protonshell/internal/servers"
)

func initCmds(m Model) tea.Cmd {
	return tea.Batch(tick(), loadList(m.listPath, time.Time{}), fetchStatus())
}

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		clampCursor(&m)
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
			rebuild(&m)
		}
		return m, nil

	case statusMsg:
		m.statusInFlight = false
		if msg.err != nil {
			setError(&m, "status: "+msg.err.Error())
			return m, nil
		}
		m.status, m.statusKnown = msg.status, true
		return m, nil

	case actionMsg:
		m.busy = ""
		if msg.err != nil {
			setError(&m, msg.err.Error())
		} else {
			m.message, m.isError = firstLine(msg.out), false
		}
		m.statusInFlight = true
		return m, fetchStatus()

	case tea.KeyPressMsg:
		if m.filter.Focused() {
			return updateFilter(m, msg)
		}
		return updateKeys(m, msg)
	}
	return m, nil
}

func updateFilter(m Model, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.filter.SetValue("")
		m.filter.Blur()
		rebuild(&m)
		return m, nil
	case "enter":
		m.filter.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	rebuild(&m)
	return m, cmd
}

func updateKeys(m Model, msg tea.KeyPressMsg) (Model, tea.Cmd) {
	page := visibleRows(m)
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
		rebuild(&m)
	case "s":
		m.sort = (m.sort + 1) % sortModes
		rebuild(&m)
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
	clampCursor(&m)
	return m, nil
}

// rebuild reapplies the filter and sort, keeping the cursor on the same
// server when it is still in the result.
func rebuild(m *Model) {
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
	slices.SortStableFunc(m.rows, func(a, b servers.Logical) int { return compareServers(m.sort, a, b) })

	m.cursor = 0
	if selected != "" {
		if i := slices.IndexFunc(m.rows, func(s servers.Logical) bool { return s.Name == selected }); i >= 0 {
			m.cursor = i
		}
	}
	clampCursor(m)
}

func matches(s servers.Logical, query string) bool {
	return strings.Contains(strings.ToLower(s.Name), query) ||
		strings.Contains(strings.ToLower(s.City), query) ||
		strings.ToLower(s.ExitCountry) == query
}

func compareServers(s sortMode, a, b servers.Logical) int {
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

func clampCursor(m *Model) {
	m.cursor = max(0, min(m.cursor, len(m.rows)-1))
	visible := visibleRows(*m)
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	m.offset = max(0, min(m.offset, len(m.rows)-visible))
}

func setError(m *Model, text string) {
	m.message, m.isError = text, true
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}
