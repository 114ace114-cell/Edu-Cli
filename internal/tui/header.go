package tui

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	borderColor = colorAccent
	accentColor = colorAccent
	dimColor    = colorDim
	infoColor   = colorInfo
)

const (
	headerTotalFrames = 12
	headerTickMs      = 80
	headerLeftWidth   = 16
)

var headerTips = []string{
	"Try /research for deep analysis",
	"Use /sessions to resume work",
	"Type /help to see all commands",
	"Use /model to switch model tier",
	"Try /swarm for multi-agent tasks",
}

func headerFrameTick() tea.Cmd {
	return tea.Tick(time.Duration(headerTickMs)*time.Millisecond, func(time.Time) tea.Msg {
		return headerTickMsg{}
	})
}

// Activity is one recent session row in the startup header.
type Activity struct {
	Title string
	When  time.Time
	Msgs  int
}

func buildTimeLabel() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	fi, err := os.Stat(exe)
	if err != nil {
		return ""
	}
	return fi.ModTime().Format("15:04")
}

func renderStartupHeader(frame, width int, version, modelTier, endpoint, cwd string, sessions []Activity, tipIdx int, agentLabel string) string {
	if width < 50 {
		width = 50
	}
	if width > 100 {
		width = 100
	}

	innerWidth := width - 2
	rightWidth := innerWidth - headerLeftWidth - 1

	leftLines := renderKocoroGrid(frame)

	var rightLines []string
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	modelStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)

	tipHeader := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Tips")
	rightLines = append(rightLines, " "+tipHeader)
	rightLines = append(rightLines, " "+dimStyle.Render(truncateStr(headerTips[tipIdx%len(headerTips)], rightWidth-3)))
	rightLines = append(rightLines, " "+dimStyle.Render(strings.Repeat("─", rightWidth-2)))

	actHeader := lipgloss.NewStyle().Foreground(infoColor).Bold(true).Render("Recent activity")
	rightLines = append(rightLines, " "+actHeader)
	if len(sessions) == 0 {
		rightLines = append(rightLines, " "+dimStyle.Render("No recent sessions"))
		rightLines = append(rightLines, "")
	} else {
		s := sessions[0]
		titleStyle := lipgloss.NewStyle().Foreground(colorSecondary)
		rightLines = append(rightLines, " "+titleStyle.Render(truncateStr(s.Title, rightWidth-4)))
		when := s.When
		if when.IsZero() {
			when = time.Now()
		}
		rightLines = append(rightLines, " "+dimStyle.Render(fmt.Sprintf("%s, %d msgs", timeAgo(when), s.Msgs)))
	}
	rightLines = append(rightLines, " "+dimStyle.Render(truncateStr(cwd, rightWidth-3)))

	info := ""
	if agentLabel != "" {
		info = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("▌ "+agentLabel) + dimStyle.Render("  ·  ")
	}
	info += modelStyle.Render(modelTier) + dimStyle.Render(" · v"+version)
	if bt := buildTimeLabel(); bt != "" {
		info += dimStyle.Render(" · built " + bt)
	}
	if budget := rightWidth - lipgloss.Width(info) - 5; budget >= 6 {
		info += dimStyle.Render(" · " + truncateStr(endpoint, budget))
	}
	rightLines = append(rightLines, " "+info)

	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	bdr := lipgloss.NewStyle().Foreground(borderColor)
	var sb strings.Builder
	titlePart := "─ Kocoro CLI "
	remaining := innerWidth - lipgloss.Width(titlePart)
	if remaining < 0 {
		remaining = 0
	}
	sb.WriteString(bdr.Render("╭"+titlePart+strings.Repeat("─", remaining)+"╮") + "\n")
	divider := bdr.Render("│")
	for i := range leftLines {
		left := padToWidth(leftLines[i], headerLeftWidth)
		right := padToWidth(rightLines[i], rightWidth)
		sb.WriteString(bdr.Render("│") + left + divider + right + bdr.Render("│") + "\n")
	}
	sb.WriteString(bdr.Render("╰" + strings.Repeat("─", innerWidth) + "╯"))
	return sb.String()
}

func padToWidth(styled string, targetWidth int) string {
	visible := lipgloss.Width(styled)
	if visible >= targetWidth {
		return styled
	}
	return styled + strings.Repeat(" ", targetWidth-visible)
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 3 {
		maxLen = 4
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	end := maxLen - 3
	if end < 1 {
		end = 1
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > end {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 48*time.Hour:
		return "yesterday"
	default:
		days := int(d.Hours() / 24)
		if days < 1 {
			days = 1
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func pickTipIdx() int { return rand.Intn(len(headerTips)) }
