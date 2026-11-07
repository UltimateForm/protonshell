package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/UltimateForm/protonshell/internal/servers"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	badStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Reverse(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
)

// A row is formatted in three parts so the load cell can be coloured alone.
const (
	rowLeft   = "%-2s%-14s %-3s %-4s %-22s "
	rowLoad   = "%5s"
	rowRight  = "  %-11s %6s"
	rowFormat = rowLeft + rowLoad + rowRight
)

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	lines := make([]string, 0, m.height)
	lines = append(lines, m.statusLine(), m.infoLine())
	lines = append(lines, headerStyle.Render(m.pad(fmt.Sprintf(rowFormat,
		"", "SERVER", "CC", "VIA", "CITY", "LOAD", "FEATURES", "SCORE"))))

	visible := m.visibleRows()
	end := min(m.offset+visible, len(m.rows))
	for i := m.offset; i < end; i++ {
		lines = append(lines, m.row(m.rows[i], i == m.cursor))
	}
	for i := end - m.offset; i < visible; i++ {
		lines = append(lines, "")
	}

	lines = append(lines, m.messageLine(), m.footerLine())
	return strings.Join(lines, "\n")
}

func (m Model) statusLine() string {
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
	return m.clip(titleStyle.Render("protonshell") + "  " + state)
}

func (m Model) infoLine() string {
	speed := dimStyle.Render("↓ –  ↑ –")
	if m.speedOK {
		speed = okStyle.Render("↓ "+humanRate(m.rx)) + "  " + warnStyle.Render("↑ "+humanRate(m.tx))
	}
	count := fmt.Sprintf("%d/%d servers", len(m.rows), len(m.all))
	return m.clip(speed + dimStyle.Render("  ·  "+count+"  ·  sort: "+m.sort.String()))
}

func (m Model) row(s servers.Logical, selected bool) string {
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
	left := fmt.Sprintf(rowLeft, marker, s.Name, s.ExitCountry, via, truncate(s.City, 22))
	load := fmt.Sprintf(rowLoad, fmt.Sprintf("%d%%", s.Load))
	right := fmt.Sprintf(rowRight, features, fmt.Sprintf("%.2f", s.Score))

	if selected {
		return selectedStyle.Render(m.pad(left + load + right))
	}
	return m.clip(left + loadStyle(s.Load).Render(load) + right)
}

func (m Model) messageLine() string {
	if m.listErr != "" {
		return m.clip(badStyle.Render(m.listErr))
	}
	if m.message == "" {
		return ""
	}
	if m.isError {
		return m.clip(badStyle.Render(m.message))
	}
	return m.clip(dimStyle.Render(m.message))
}

func (m Model) footerLine() string {
	if m.filter.Focused() {
		return m.filter.View()
	}
	help := "↑↓ move · enter connect · d disconnect · / filter · s sort · r refresh · q quit"
	if v := m.filter.Value(); v != "" {
		help = "filter: " + v + " (esc clears) · " + help
	}
	return m.clip(dimStyle.Render(help))
}

func loadStyle(load int) lipgloss.Style {
	switch {
	case load < 50:
		return okStyle
	case load < 80:
		return warnStyle
	default:
		return badStyle
	}
}

// pad fits plain text to exactly the terminal width.
func (m Model) pad(s string) string {
	return lipgloss.NewStyle().Width(m.width).MaxWidth(m.width).Render(s)
}

// clip cuts styled text at the terminal width so it never wraps.
func (m Model) clip(s string) string {
	return lipgloss.NewStyle().MaxWidth(m.width).Render(s)
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
