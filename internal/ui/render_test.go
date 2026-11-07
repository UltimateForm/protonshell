package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/UltimateForm/protonshell/internal/servers"
)

func TestRowIndexes(t *testing.T) {
	var list []servers.Logical
	for i := range 20 {
		list = append(list, srv(fmt.Sprintf("PT#%d", i+1), "PT", 40, 1.0+float64(i)/100))
	}
	list = append(list, srv("ES#1", "ES", 40, 2), srv("FR#1", "FR", 40, 3))
	m := newTestModel(list...)
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10}) // 5 visible rows
	m = press(m, tea.KeyRight)
	m = press(m, tea.KeyEnd)

	lines := strings.Split(ansi.Strip(render(m)), "\n")
	tableTop := 3 // status, info, header
	wantRows := []struct{ country, server string }{
		{"  1 PT ", "  16 PT#16 "},
		{"  2 ES ", "  17 PT#17 "},
		{"  3 FR ", "  18 PT#18 "},
		{"", "  19 PT#19 "},
		{"", "  20 PT#20 "},
	}
	for i, want := range wantRows {
		line := lines[tableTop+i]
		if want.country != "" && !strings.HasPrefix(line, want.country) {
			t.Errorf("line %d = %q, want country prefix %q", i, line, want.country)
		}
		if !strings.Contains(line, want.server) {
			t.Errorf("line %d = %q, want server cell %q", i, line, want.server)
		}
	}
}

func TestFitTruncatesInsteadOfWrapping(t *testing.T) {
	tests := []struct {
		in    string
		width int
		want  string
	}{
		{"abcdefgh", 5, "abcde"},
		{"abc defgh", 5, "abc d"},
		{"ab", 5, "ab   "},
	}
	for _, tt := range tests {
		if got := ansi.Strip(fit(tt.width, tt.in)); got != tt.want {
			t.Errorf("fit(%d, %q) = %q, want %q", tt.width, tt.in, got, tt.want)
		}
	}
}

func TestRenderFillsExactlyTheScreen(t *testing.T) {
	m := newTestModel(srv("PT#1", "PT", 40, 1), srv("ES#1", "ES", 40, 2))
	// Narrower than the status line hits the TODO in statusLine.
	for _, width := range []int{120, 90, 60} {
		m, _ = update(m, tea.WindowSizeMsg{Width: width, Height: 10})
		lines := strings.Split(render(m), "\n")
		if len(lines) != 10 {
			t.Errorf("width %d: render produced %d lines, want 10", width, len(lines))
		}
		for i, line := range lines {
			if w := ansi.StringWidth(line); w > width {
				t.Errorf("width %d: line %d is %d cells wide", width, i, w)
			}
		}
	}
}

func TestInfoLineCountsSelectedCountry(t *testing.T) {
	m := newTestModel(
		srv("PT#1", "PT", 40, 1), srv("PT#2", "PT", 40, 1), srv("PT#3", "PT", 40, 1),
		srv("ES#1", "ES", 40, 2),
	)
	if got, want := ansi.Strip(infoLine(m)), "3/4 servers"; !strings.Contains(got, want) {
		t.Errorf("info line = %q, want it to contain %q", got, want)
	}
	m = press(m, tea.KeyDown) // ES
	if got, want := ansi.Strip(infoLine(m)), "1/4 servers"; !strings.Contains(got, want) {
		t.Errorf("info line after down = %q, want it to contain %q", got, want)
	}
}
