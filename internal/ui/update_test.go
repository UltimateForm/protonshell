package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/UltimateForm/protonshell/internal/servers"
)

func srv(name, cc string, load int, score float64) servers.Logical {
	return servers.Logical{Name: name, ExitCountry: cc, EntryCountry: cc, Load: load, Score: score}
}

// newTestModel returns a sized model holding the given servers.
func newTestModel(list ...servers.Logical) Model {
	m := New("", "lo", "test")
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m.all = list
	rebuild(&m)
	return m
}

func press(m Model, code rune) Model {
	m, _ = updateKeys(m, tea.KeyPressMsg{Code: code})
	return m
}

func names(rows []servers.Logical) []string {
	out := make([]string, len(rows))
	for i, s := range rows {
		out[i] = s.Name
	}
	return out
}

func TestGroupCountries(t *testing.T) {
	got := groupCountries([]servers.Logical{
		srv("PT#1", "PT", 40, 1.9),
		srv("ES#1", "ES", 10, 1.5),
		srv("PT#2", "PT", 60, 1.2),
	})
	want := []country{
		{Code: "PT", Servers: 2, Best: 1.2, AvgLoad: 50},
		{Code: "ES", Servers: 1, Best: 1.5, AvgLoad: 10},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d countries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("country %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDefaultSortIsCost(t *testing.T) {
	m := newTestModel(
		srv("AT#1", "AT", 10, 3.0),
		// Cost, load and name each order PT's servers differently.
		srv("PT#1", "PT", 10, 2.0),
		srv("PT#2", "PT", 50, 1.5),
		srv("ES#1", "ES", 10, 2.5),
	)
	var codes []string
	for _, c := range m.countries {
		codes = append(codes, c.Code)
	}
	if want := []string{"PT", "ES", "AT"}; len(codes) != 3 || codes[0] != want[0] || codes[1] != want[1] || codes[2] != want[2] {
		t.Fatalf("countries = %v, want %v (lowest cost first)", codes, want)
	}
	if got := names(m.rows); len(got) != 2 || got[0] != "PT#2" || got[1] != "PT#1" {
		t.Fatalf("PT servers = %v, want [PT#2 PT#1] (lowest cost first)", got)
	}
}

func TestCountryCursorSwapsServerPane(t *testing.T) {
	m := newTestModel(
		srv("PT#1", "PT", 10, 1.0),
		srv("ES#2", "ES", 10, 1.1),
		srv("ES#1", "ES", 10, 1.0),
	)
	// Countries sort by code by default: ES, PT.
	if got := names(m.rows); len(got) != 2 || got[0] != "ES#1" || got[1] != "ES#2" {
		t.Fatalf("initial rows = %v, want [ES#1 ES#2]", got)
	}
	m = press(m, tea.KeyDown)
	if got := names(m.rows); len(got) != 1 || got[0] != "PT#1" {
		t.Fatalf("rows after down = %v, want [PT#1]", got)
	}
}

func TestServerPaneNavigation(t *testing.T) {
	m := newTestModel(srv("ES#1", "ES", 10, 1.0), srv("ES#2", "ES", 10, 1.1), srv("PT#1", "PT", 10, 1.0))
	m = press(m, tea.KeyRight)
	if m.focus != paneServers {
		t.Fatalf("focus after right = %v, want servers", m.focus)
	}
	m = press(m, tea.KeyDown)
	if m.cursor != 1 || m.countryCursor != 0 {
		t.Fatalf("after down in server pane: cursor=%d countryCursor=%d, want 1 and 0", m.cursor, m.countryCursor)
	}
	m = press(m, tea.KeyLeft)
	if m.focus != paneCountries {
		t.Fatalf("focus after left = %v, want countries", m.focus)
	}
}

func TestJumpKeys(t *testing.T) {
	var list []servers.Logical
	for i := range 50 {
		list = append(list, srv("ES#"+string(rune('A'+i)), "ES", 10, float64(i)))
	}
	m := newTestModel(list...) // height 20 → visibleRows 15
	m = press(m, tea.KeyRight)

	m = press(m, tea.KeyPgDown)
	if m.cursor != 15 {
		t.Fatalf("cursor after pgdown = %d, want 15", m.cursor)
	}
	m = press(m, tea.KeyEnd)
	if m.cursor != 49 || m.offset != 35 {
		t.Fatalf("after end: cursor=%d offset=%d, want 49 and 35", m.cursor, m.offset)
	}
	m = press(m, tea.KeyPgUp)
	if m.cursor != 34 {
		t.Fatalf("cursor after pgup = %d, want 34", m.cursor)
	}
	m = press(m, tea.KeyHome)
	if m.cursor != 0 || m.offset != 0 {
		t.Fatalf("after home: cursor=%d offset=%d, want 0 and 0", m.cursor, m.offset)
	}
}

func TestFilterKeepsSelectedCountry(t *testing.T) {
	m := newTestModel(srv("ES#1", "ES", 10, 1.0), srv("PT#1", "PT", 10, 1.0), srv("US#1", "US", 10, 1.0))
	m = press(m, tea.KeyDown) // PT
	m.filter.SetValue("#1")   // matches every server
	rebuild(&m)
	if got := selectedCountry(m); got != "PT" {
		t.Fatalf("selected country after filter = %q, want PT", got)
	}
}

func TestEnterOnCountryConnectsByCountry(t *testing.T) {
	m := newTestModel(srv("ES#1", "ES", 10, 1.0))
	m, cmd := updateKeys(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on country pane returned no command")
	}
	if want := "Connecting to fastest server in ES…"; m.busy != want {
		t.Fatalf("busy = %q, want %q", m.busy, want)
	}
}
