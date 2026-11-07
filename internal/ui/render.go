package ui

import (
	"fmt"
	"strings"

	lp "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	titleStyle    = lp.NewStyle().Bold(true).Foreground(lp.Color("13"))
	dimStyle      = lp.NewStyle().Foreground(lp.Color("8"))
	okStyle       = lp.NewStyle().Foreground(lp.Color("10"))
	warnStyle     = lp.NewStyle().Foreground(lp.Color("11"))
	badStyle      = lp.NewStyle().Foreground(lp.Color("9"))
	headerStyle   = lp.NewStyle().Bold(true).Reverse(true)
	selectedStyle = lp.NewStyle().Reverse(true)
	// unfocusedStyle marks the cursor row of the pane without focus.
	unfocusedStyle = lp.NewStyle().Background(lp.Color("8"))
)

// A server row is formatted in parts so the index and load cells can be
// styled alone. There is no country column: the pane shows one country.
const (
	rowMarker = "%-2s"
	rowIndex  = "%4s " // 1-based position in the pane, no heading
	rowLeft   = "%-14s %-4s %-22s "
	rowLoad   = "%5s"
	rowRight  = "  %-11s %6s"
	rowFormat = rowMarker + rowIndex + rowLeft + rowLoad + rowRight
)

// A country row is formatted in parts so the index cell can be styled alone.
const (
	countryIndex  = "%3s " // 1-based position in the pane, no heading
	countryRest   = "%-3s %7s  %5s  %4s"
	countryFormat = countryIndex + countryRest
	// countryPaneWidth fits countryFormat plus a trailing space.
	countryPaneWidth = 29
	paneSeparator    = " "
)

func render(m Model) string {
	if m.width == 0 {
		return ""
	}
	// TODO: responsive layout: the panes are fixed-width (29 + 1 + 75 cols). Below
	// ~105 cols the server pane is cut off on the right; consider dropping columns
	// (FEATURES, VIA, CITY) or stacking the panes on narrow terminals.
	rightWidth := max(1, m.width-countryPaneWidth-len(paneSeparator))

	lines := make([]string, 0, m.height)
	lines = append(lines, statusLine(m), infoLine(m))
	lines = append(lines, clip(m,
		headerStyle.Render(fit(countryPaneWidth, fmt.Sprintf(countryFormat, "", "CC", "SERVERS", "COST", "LOAD")))+
			paneSeparator+
			headerStyle.Render(fit(rightWidth, fmt.Sprintf(rowFormat, "", "", "SERVER", "VIA", "CITY", "LOAD", "FEATURES", "COST")))))

	for i := range visibleRows(m) {
		left := countryCell(m, m.countryOffset+i)
		right := serverCell(m, m.offset+i, rightWidth)
		lines = append(lines, clip(m, left+paneSeparator+right))
	}

	lines = append(lines, messageLine(m), footerLine(m))
	return strings.Join(lines, "\n")
}

// cursorStyle picks how a row is drawn given whether it holds the cursor and
// whether its pane has focus.
func cursorStyle(m Model, p pane, atCursor bool) (lp.Style, bool) {
	switch {
	case !atCursor:
		return lp.Style{}, false
	case m.focus == p:
		return selectedStyle, true
	default:
		return unfocusedStyle, true
	}
}

func countryCell(m Model, i int) string {
	if i >= len(m.countries) {
		return fit(countryPaneWidth, "")
	}
	c := m.countries[i]
	index := fmt.Sprintf(countryIndex, fmt.Sprint(i+1))
	rest := fmt.Sprintf(countryRest, c.Code,
		fmt.Sprint(c.Servers), fmt.Sprintf("%.2f", c.Best), fmt.Sprintf("%d%%", c.AvgLoad))
	if style, ok := cursorStyle(m, paneCountries, i == m.countryCursor); ok {
		return style.Render(fit(countryPaneWidth, index+rest))
	}
	return fit(countryPaneWidth, dimStyle.Render(index)+rest)
}

func serverCell(m Model, i, width int) string {
	if i >= len(m.rows) {
		return ""
	}
	return row(m, i, i == m.cursor, width)
}

func statusLine(m Model) string {
	var state string
	switch {
	case m.busy != "":
		state = warnStyle.Render(m.busy)
	case !m.statusKnown:
		state = dimStyle.Render("checking status…")
	case m.status.Connected:
		state = okStyle.Render("● Connected") + " " + m.status.Server
		if m.status.Location != "" {
			state += dimStyle.Render(" · " + m.status.Location)
		}
		if m.status.Protocol != "" {
			state += dimStyle.Render(" · " + m.status.Protocol)
		}
	default:
		state = badStyle.Render("○ Disconnected")
	}
	statusLeft := titleStyle.Render("protonshell") + "  " + state
	statusRight := lp.NewStyle().Foreground(lp.Darken(lp.White, 0.5)).Render(m.version)
	// TODO: fix bug: padding goes negative when the terminal is narrower than
	// statusLeft+statusRight, and strings.Repeat below panics. Clamp with max(0, padding).
	padding := m.width - lp.Width(statusLeft) - lp.Width(statusRight)
	return clip(m, statusLeft+strings.Repeat(" ", padding)+statusRight)
}

func infoLine(m Model) string {
	speed := dimStyle.Render("↓ –  ↑ –")
	if m.speedOK {
		speed = okStyle.Render("↓ "+humanRate(m.rx)) + "  " + warnStyle.Render("↑ "+humanRate(m.tx))
	}
	// Servers in the selected country (after the filter) out of all servers.
	count := fmt.Sprintf("%d countries  ·  %d/%d servers", len(m.countries), len(m.rows), len(m.all))
	sort := "sort countries: " + m.countrySort.String()
	if m.focus == paneServers {
		sort = "sort servers: " + m.sort.String()
	}
	return clip(m, speed+dimStyle.Render("  ·  "+count+"  ·  "+sort))
}

func row(m Model, i int, atCursor bool, width int) string {
	s := m.rows[i]
	marker := ""
	if m.status.Connected && s.Name == m.status.Server {
		marker = "●"
	}
	via := ""
	if s.EntryCountry != s.ExitCountry {
		via = s.EntryCountry
	}
	features := s.Features.String()
	if s.Tier == 0 {
		features = strings.TrimPrefix(features+",FREE", ",")
	}
	marker = fmt.Sprintf(rowMarker, marker)
	index := fmt.Sprintf(rowIndex, fmt.Sprint(i+1))
	left := fmt.Sprintf(rowLeft, s.Name, via, truncate(s.City, 22))
	load := fmt.Sprintf(rowLoad, fmt.Sprintf("%d%%", s.Load))
	right := fmt.Sprintf(rowRight, features, fmt.Sprintf("%.2f", s.Score))

	if style, ok := cursorStyle(m, paneServers, atCursor); ok {
		return style.Render(fit(width, marker+index+left+load+right))
	}
	return fit(width, marker+dimStyle.Render(index)+left+loadStyle(s.Load).Render(load)+right)
}

func messageLine(m Model) string {
	if m.listErr != "" {
		return clip(m, badStyle.Render(m.listErr))
	}
	if m.message == "" {
		return ""
	}
	if m.isError {
		return clip(m, badStyle.Render(m.message))
	}
	return clip(m, dimStyle.Render(m.message))
}

func footerLine(m Model) string {
	if m.filter.Focused() {
		return m.filter.View()
	}
	help := "←→ pane · ↑↓ move · pgup/pgdn page · g/G top/bottom · enter connect · d disconnect · / filter · s sort · r refresh · q quit"
	if v := m.filter.Value(); v != "" {
		help = "filter: " + v + " (esc clears) · " + help
	}
	return clip(m, dimStyle.Render(help))
}

func loadStyle(load int) lp.Style {
	switch {
	case load < 50:
		return okStyle
	case load < 80:
		return warnStyle
	default:
		return badStyle
	}
}

// fit cuts or pads text to exactly width cells. It truncates before padding
// because lipgloss's Width wraps overlong text onto extra lines.
func fit(width int, s string) string {
	return lp.NewStyle().Width(width).Render(ansi.Truncate(s, width, ""))
}

// clip cuts styled text at the terminal width so it never wraps.
func clip(m Model, s string) string {
	return lp.NewStyle().MaxWidth(m.width).Render(s)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func humanRate(bytesPerSec float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	i := 0
	for bytesPerSec >= 1024 && i < len(units)-1 {
		bytesPerSec /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", bytesPerSec, units[i])
	}
	return fmt.Sprintf("%.1f %s", bytesPerSec, units[i])
}
