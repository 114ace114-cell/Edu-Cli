package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// shipMask is a 16×16 side-view ship: sail, mast, and hull.
var kocoroMask = [16]string{
	"0000000000000000",
	"0000001000000000",
	"0000001100000000",
	"0000001110000000",
	"0000001111000000",
	"0000001111100000",
	"0000001000000000",
	"0000001000000000",
	"0000001000000000",
	"0011111111100000",
	"0111111111111110",
	"1111111111111111",
	"0111111111111110",
	"0011111111111100",
	"0000000000000000",
	"0000000000000000",
}

const kocoroSize = 16

var (
	kocoroFrom = [3]float64{12, 92, 48}
	kocoroTo   = [3]float64{186, 255, 140}
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
