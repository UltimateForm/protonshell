package ui

func visibleRows(m Model) int {
	return max(1, m.height-chromeLines)
}
