package tui

import "github.com/charmbracelet/lipgloss"

// Dark-terminal palette matching internal/tui/theme.go. The full TUI probes
// the background and swaps AdaptiveColor; this screen is the dark look from
// the Kocoro CLI home frame.
var (
	colorDim       = lipgloss.Color("245")
	colorSecondary = lipgloss.Color("252")
	colorFaint     = lipgloss.Color("237")
	colorError     = lipgloss.Color("196")
	colorWarn      = lipgloss.Color("214")
	colorAccent    = lipgloss.Color("#FF5C8A")
	colorInfo      = lipgloss.Color("39")
)

func styleDim() lipgloss.Style       { return lipgloss.NewStyle().Foreground(colorDim) }
func styleSecondary() lipgloss.Style { return lipgloss.NewStyle().Foreground(colorSecondary) }
func styleFaint() lipgloss.Style     { return lipgloss.NewStyle().Foreground(colorFaint) }
func styleError() lipgloss.Style     { return lipgloss.NewStyle().Foreground(colorError) }
func styleWarn() lipgloss.Style      { return lipgloss.NewStyle().Foreground(colorWarn) }
