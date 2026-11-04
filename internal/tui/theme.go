package tui

import "github.com/charmbracelet/lipgloss"

// Dark-terminal palette. Accent and info are both green so the frame,
// prompt, and status line share one look.
var (
	colorDim       = lipgloss.Color("245")
	colorSecondary = lipgloss.Color("252")
	colorFaint     = lipgloss.Color("237")
	colorError     = lipgloss.Color("196")
	colorWarn      = lipgloss.Color("214")
	colorAccent    = lipgloss.Color("#22C55E")
	colorInfo      = lipgloss.Color("#86EFAC")
)

func styleDim() lipgloss.Style       { return lipgloss.NewStyle().Foreground(colorDim) }
func styleSecondary() lipgloss.Style { return lipgloss.NewStyle().Foreground(colorSecondary) }
func styleFaint() lipgloss.Style     { return lipgloss.NewStyle().Foreground(colorFaint) }
func styleError() lipgloss.Style     { return lipgloss.NewStyle().Foreground(colorError) }
func styleWarn() lipgloss.Style      { return lipgloss.NewStyle().Foreground(colorWarn) }
