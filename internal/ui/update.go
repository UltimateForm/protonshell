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
		clampCountryCursor(&m)
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
	// far is at least as long as either table, for jumps to either end.
	far := len(m.all) + 1
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "left", "h":
		m.focus = paneCountries
	case "right", "l":
		if len(m.rows) > 0 {
			m.focus = paneServers
		}
	case "up", "k":
		moveCursor(&m, -1)
	case "down", "j":
		moveCursor(&m, 1)
	case "pgup":
		moveCursor(&m, -page)
	case "pgdown":
		moveCursor(&m, page)
	case "home", "g":
		moveCursor(&m, -far)
	case "end", "G":
		moveCursor(&m, far)
	case "/":
		return m, m.filter.Focus()
	case "esc":
		m.filter.SetValue("")
		rebuild(&m)
	case "s":
		if m.focus == paneCountries {
			m.countrySort = (m.countrySort + 1) % countrySortModes
		} else {
			m.sort = (m.sort + 1) % sortModes
		}
		rebuild(&m)
	case "r":
		if !m.statusInFlight {
			m.statusInFlight = true
			return m, fetchStatus()
		}
	case "enter":
		if m.busy != "" {
			return m, nil
		}
		if m.focus == paneCountries {
			if len(m.countries) == 0 {
				return m, nil
			}
			code := m.countries[m.countryCursor].Code
			m.busy = "Connecting to fastest server in " + code + "…"
			return m, connectCountry(code)
		}
		if len(m.rows) == 0 {
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
	return m, nil
}

// moveCursor moves the cursor of the focused pane. Moving between countries
// swaps the server pane's contents.
func moveCursor(m *Model, delta int) {
	if m.focus == paneServers {
		m.cursor += delta
		clampCursor(m)
		return
	}
	before := m.countryCursor
	m.countryCursor += delta
	clampCountryCursor(m)
	if m.countryCursor != before {
		rebuildRows(m)
	}
}

// rebuild reapplies the filter and both sorts, keeping each cursor on the
// same country and server when they are still in the result.
func rebuild(m *Model) {
	selected := selectedCountry(*m)

	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.matched = m.matched[:0]
	for _, s := range m.all {
		if query == "" || matches(s, query) {
			m.matched = append(m.matched, s)
		}
	}

	m.countries = groupCountries(m.matched)
	slices.SortStableFunc(m.countries, func(a, b country) int { return compareCountries(m.countrySort, a, b) })
	m.countryCursor = 0
	if selected != "" {
		if i := slices.IndexFunc(m.countries, func(c country) bool { return c.Code == selected }); i >= 0 {
			m.countryCursor = i
		}
	}
	clampCountryCursor(m)
	rebuildRows(m)
}

// rebuildRows fills the server pane with the matched servers of the selected
// country, keeping the cursor on the same server when it is still there.
func rebuildRows(m *Model) {
	var selected string
	if m.cursor < len(m.rows) {
		selected = m.rows[m.cursor].Name
	}

	code := selectedCountry(*m)
	m.rows = m.rows[:0]
	for _, s := range m.matched {
		if s.ExitCountry == code {
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
	if len(m.rows) == 0 {
		m.focus = paneCountries
	}
}

func selectedCountry(m Model) string {
	if m.countryCursor < len(m.countries) {
		return m.countries[m.countryCursor].Code
	}
	return ""
}

// groupCountries summarises servers per ExitCountry, in order of first
// appearance.
func groupCountries(list []servers.Logical) []country {
	index := map[string]int{}
	var out []country
	var loadSums []int
	for _, s := range list {
		i, ok := index[s.ExitCountry]
		if !ok {
			i = len(out)
			index[s.ExitCountry] = i
			out = append(out, country{Code: s.ExitCountry, Best: s.Score})
			loadSums = append(loadSums, 0)
		}
		c := &out[i]
		c.Servers++
		c.Best = min(c.Best, s.Score)
		loadSums[i] += s.Load
	}
	for i := range out {
		out[i].AvgLoad = loadSums[i] / out[i].Servers
	}
	return out
}

func matches(s servers.Logical, query string) bool {
	return strings.Contains(strings.ToLower(s.Name), query) ||
		strings.Contains(strings.ToLower(s.City), query) ||
		strings.ToLower(s.ExitCountry) == query
}

func compareCountries(s countrySortMode, a, b country) int {
	switch s {
	case countrySortServers:
		return cmp.Or(cmp.Compare(b.Servers, a.Servers), cmp.Compare(a.Code, b.Code))
	case countrySortBest:
		return cmp.Or(cmp.Compare(a.Best, b.Best), cmp.Compare(a.Code, b.Code))
	case countrySortLoad:
		return cmp.Or(cmp.Compare(a.AvgLoad, b.AvgLoad), cmp.Compare(a.Code, b.Code))
	default:
		return cmp.Compare(a.Code, b.Code)
	}
}

func compareServers(s sortMode, a, b servers.Logical) int {
	switch s {
	case sortLoad:
		return cmp.Or(cmp.Compare(a.Load, b.Load), compareNames(a.Name, b.Name))
	case sortName:
		return compareNames(a.Name, b.Name)
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
	m.cursor, m.offset = clampScroll(m.cursor, m.offset, len(m.rows), visibleRows(*m))
}

func clampCountryCursor(m *Model) {
	m.countryCursor, m.countryOffset = clampScroll(m.countryCursor, m.countryOffset, len(m.countries), visibleRows(*m))
}

// clampScroll keeps cursor inside n rows and offset such that the cursor is
// within the visible window.
func clampScroll(cursor, offset, n, visible int) (int, int) {
	cursor = max(0, min(cursor, n-1))
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+visible {
		offset = cursor - visible + 1
	}
	offset = max(0, min(offset, n-visible))
	return cursor, offset
}

func setError(m *Model, text string) {
	m.message, m.isError = text, true
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}
