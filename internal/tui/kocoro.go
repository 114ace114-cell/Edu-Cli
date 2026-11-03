package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// kocoroMask is the 16×16 brand mark from internal/tui/kocoro.go.
var kocoroMask = [16]string{
	"0000000000000000",
	"0000000011100000",
	"0000000111111000",
	"0000001111111000",
	"0000011111111000",
	"0000111111111000",
	"0001111001011000",
	"0001111011100000",
	"0001111011110000",
	"0001111011111000",
	"0000111101111000",
	"0000011101111000",
	"0000000011111000",
	"0000000111111000",
	"0000000011100000",
	"0000000000000000",
}

const kocoroSize = 16

var (
	kocoroFrom = [3]float64{244, 7, 82}
	kocoroTo   = [3]float64{249, 171, 143}
)

func kocoroInk(row, col int) bool { return kocoroMask[row][col] == '1' }

func kocoroColor(col, row, frame int) lipgloss.Color {
	t := float64(col+row)/float64(2*(kocoroSize-1)) + float64(frame)/24.0
	t -= float64(int(t))
	r := kocoroFrom[0] + (kocoroTo[0]-kocoroFrom[0])*t
	g := kocoroFrom[1] + (kocoroTo[1]-kocoroFrom[1])*t
	b := kocoroFrom[2] + (kocoroTo[2]-kocoroFrom[2])*t
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", int(r), int(g), int(b)))
}

func renderKocoroGrid(frame int) []string {
	revealRows := 6 + frame*2
	lines := make([]string, kocoroSize/2)
	for i := range lines {
		top, bot := i*2, i*2+1
		var sb strings.Builder
		for col := 0; col < kocoroSize; col++ {
			tInk := kocoroInk(top, col) && top < revealRows
			bInk := kocoroInk(bot, col) && bot < revealRows
			switch {
			case !tInk && !bInk:
				sb.WriteByte(' ')
			case tInk && !bInk:
				sb.WriteString(lipgloss.NewStyle().Foreground(kocoroColor(col, top, frame)).Render("▀"))
			case !tInk && bInk:
				sb.WriteString(lipgloss.NewStyle().Foreground(kocoroColor(col, bot, frame)).Render("▄"))
			default:
				sb.WriteString(lipgloss.NewStyle().
					Foreground(kocoroColor(col, top, frame)).
					Background(kocoroColor(col, bot, frame)).Render("▀"))
			}
		}
		lines[i] = sb.String()
	}
	return lines
}
