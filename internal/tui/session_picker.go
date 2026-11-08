package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const dropListSize = 8

func renderDropList(maxVisible, total, selected int, item func(i int) (label, desc string)) string {
	if total == 0 {
		return styleDim().Render("  No saved sessions") + "\n"
	}
	visible := total
	if visible > maxVisible {
		visible = maxVisible
	}
	start := 0
	if selected >= maxVisible {
		start = selected - maxVisible + 1
	}
	if start+visible > total {
		start = total - visible
	}
	if start < 0 {
		start = 0
	}
	var sb strings.Builder
	for i := start; i < start+visible; i++ {
		label, desc := item(i)
		if i == selected {
			sb.WriteString(fmt.Sprintf("  > %-28s  %s\n",
				lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(label),
				styleSecondary().Render(desc)))
		} else {
			sb.WriteString(fmt.Sprintf("    %-28s  %s\n", label, styleDim().Render(desc)))
		}
	}
	return sb.String()
}
