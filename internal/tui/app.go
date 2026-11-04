package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	stateStartup = iota
	stateInput
	stateProcessing
)

const inputBorderOverhead = 3

// Options is the home screen plus the runner callback.
type Options struct {
	Agent     string
	Model     string
	Endpoint  string
	CWD       string
	Version   string
	Connected bool
	Activity  []Activity
	// Submit runs one turn and returns the assistant reply.
	Submit func(text string) (string, error)
}

type headerTickMsg struct{}
type spinMsg struct{}
type doneMsg struct {
	reply string
	err   error
}

// Model is the alt-screen home frame from internal/tui/app.go: animated header,
// then the same header committed above a composer and status line.
type Model struct {
	opt        Options
	state      int
	width      int
	height     int
	frame      int
	tipIdx     int
	headerDone bool
	help       bool
	ctrlCArmed bool
	viewport   viewport.Model
	textarea   textarea.Model
	history    string
	spin       int
	started    time.Time
	sessionID  string
}

// Start opens the full-screen CLI. The line reader in Run stays for pipes.
func Start(opt Options) error {
	if opt.Version == "" {
		opt.Version = "dev"
	}
	if opt.Agent == "" {
		opt.Agent = "default"
	}
	p := tea.NewProgram(newModel(opt))
	_, err := p.Run()
	return err
}

func newModel(opt Options) *Model {
	width := 80
	ta := textarea.New()
	ta.Placeholder = "Ask Kocoro anything…"
	promptStyle := lipgloss.NewStyle().Foreground(colorInfo)
	ta.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return promptStyle.Render("> ")
		}
		return "  "
	})
	ta.Focus()
	ta.SetHeight(1)
	ta.SetWidth(width - inputBorderOverhead)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(colorAccent)

	return &Model{
		opt:      opt,
		tipIdx:   pickTipIdx(),
		viewport: viewport.New(width, 20),
		textarea: ta,
		width:    width,
		height:   24,
	}
}

func (m *Model) Init() tea.Cmd {
	m.state = stateStartup
	return tea.Batch(tea.EnterAltScreen, textarea.Blink, headerFrameTick())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < 20 {
			m.width = 20
		}
		m.textarea.SetWidth(m.width - inputBorderOverhead)
		m.relayout()
		return m, nil
	case headerTickMsg:
		if m.state != stateStartup {
			return m, nil
		}
		if m.frame < headerTotalFrames-1 {
			m.frame++
			return m, headerFrameTick()
		}
		return m, m.finishHeader()
	case spinMsg:
		if m.state != stateProcessing {
			return m, nil
		}
		m.spin++
		return m, spinTick()
	case doneMsg:
		m.state = stateInput
		m.textarea.Focus()
		if msg.err != nil {
			m.append(styleError().Render("  "+msg.err.Error()) + "\n")
		} else if strings.TrimSpace(msg.reply) != "" {
			m.append(wrapReply(msg.reply, m.width))
		}
		m.append("")
		return m, textarea.Blink
	case tea.KeyMsg:
		if m.state == stateStartup && msg.Type != tea.KeyCtrlC {
			m.frame = headerTotalFrames - 1
			return m, m.finishHeader()
		}
		if msg.Type != tea.KeyCtrlC {
			m.ctrlCArmed = false
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.state == stateStartup || m.ctrlCArmed {
				return m, tea.Quit
			}
			m.ctrlCArmed = true
			return m, nil
		case tea.KeyEsc:
			if m.help {
				m.help = false
				m.relayout()
			}
			return m, nil
		}
		if m.state != stateInput {
			return m, nil
		}
		if msg.String() == "?" && strings.TrimSpace(m.textarea.Value()) == "" {
			m.help = !m.help
			m.relayout()
			return m, nil
		}
		if msg.Type == tea.KeyEnter {
			text := strings.TrimSpace(m.textarea.Value())
			m.textarea.Reset()
			if text == "" {
				return m, nil
			}
			if text == "exit" || text == "quit" {
				return m, tea.Quit
			}
			if text == "?" || text == "/help" || text == "help" {
				m.help = true
				m.relayout()
				return m, nil
			}
			m.help = false
			m.append(renderUserMessage(text, m.width))
			m.append("")
			m.state = stateProcessing
			m.started = time.Now()
			m.relayout()
			return m, tea.Batch(m.run(text), spinTick())
		}
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	if m.state == stateInput {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) run(text string) tea.Cmd {
	submit := m.opt.Submit
	return func() tea.Msg {
		if submit == nil {
			return doneMsg{err: fmt.Errorf("no runner")}
		}
		reply, err := submit(text)
		return doneMsg{reply: reply, err: err}
	}
}

func spinTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

func (m *Model) finishHeader() tea.Cmd {
	if m.headerDone {
		return nil
	}
	m.headerDone = true
	m.state = stateInput
	header := renderStartupHeader(headerTotalFrames-1, m.width, m.opt.Version, m.modelLabel(), m.opt.Endpoint, m.opt.CWD, m.opt.Activity, m.tipIdx, m.opt.Agent)
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	if m.opt.Connected {
		b.WriteString("  Connected to " + m.opt.Endpoint + "\n")
	} else if m.opt.Endpoint != "" {
		b.WriteString("  Warning: API unreachable at " + m.opt.Endpoint + "\n")
	}
	b.WriteString("\n")
	m.history = b.String()
	m.viewport.SetContent(m.history)
	m.viewport.GotoBottom()
	m.relayout()
	return textarea.Blink
}

func (m *Model) append(block string) {
	if m.history != "" && !strings.HasSuffix(m.history, "\n") {
		m.history += "\n"
	}
	m.history += block
	if !strings.HasSuffix(m.history, "\n") {
		m.history += "\n"
	}
	m.viewport.SetContent(m.history)
	m.viewport.GotoBottom()
}

func (m *Model) relayout() {
	bottomH := lipgloss.Height(m.bottomRegion())
	h := m.height - bottomH - 1
	if h < 1 {
		h = 1
	}
	m.viewport.Width = m.width
	m.viewport.Height = h
}

func (m *Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	if m.state == stateStartup {
		return lipgloss.NewStyle().Width(m.width).Height(m.height).MaxHeight(m.height).Render(
			renderStartupHeader(m.frame, m.width, m.opt.Version, m.modelLabel(), m.opt.Endpoint, m.opt.CWD, m.opt.Activity, m.tipIdx, m.opt.Agent))
	}
	return m.viewport.View() + "\n" + m.bottomRegion()
}

func (m *Model) bottomRegion() string {
	var sb strings.Builder
	if m.help && m.state == stateInput {
		sb.WriteString(styleDim().Render("  ? toggle commands    exit leave    Ctrl+C twice quit"))
		sb.WriteString("\n")
	}
	switch m.state {
	case stateProcessing:
		sb.WriteString(renderInputBox(styleDim().Render("> "), m.width))
	default:
		sb.WriteString(renderInputBox(m.textarea.View(), m.width))
	}
	sb.WriteString("\n")
	marker := lipgloss.NewStyle().Foreground(colorAccent).Render("▌")
	agentSeg := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(m.opt.Agent)
	modelSeg := styleSecondary().Render(m.modelLabel())
	left := " " + marker + " " + agentSeg + " " + styleDim().Render("·") + " " + modelSeg
	right := styleDim().Render("? for commands")
	if m.ctrlCArmed {
		right = styleWarn().Render("Press Ctrl+C again to exit")
	}
	if m.state == stateProcessing {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		glyph := frames[m.spin%len(frames)]
		label := "Thinking"
		elapsed := time.Since(m.started).Truncate(time.Second).String()
		left = " " + lipgloss.NewStyle().Foreground(colorAccent).Render(glyph) + " " + styleDim().Render(label)
		right = styleDim().Render(m.modelLabel() + " " + elapsed)
	}
	sb.WriteString(composeBar(m.width-1, left, right))
	return strings.TrimRight(sb.String(), "\n")
}

func (m *Model) modelLabel() string {
	model := m.opt.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	ep := strings.ToLower(m.opt.Endpoint)
	if (strings.Contains(ep, "11434") || strings.Contains(ep, "ollama")) && !strings.Contains(model, "/") {
		return "ollama/" + model
	}
	return model
}

func renderInputBox(taView string, totalWidth int) string {
	if totalWidth < 4 {
		return taView
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Width(totalWidth - inputBorderOverhead).
		Render(taView)
}

func composeBar(width int, left, right string) string {
	if width < 0 {
		width = 0
	}
	fill := width - lipgloss.Width(left) - lipgloss.Width(right)
	if fill < 0 {
		return styleFaint().Render(strings.Repeat("─", width))
	}
	return left + styleFaint().Render(strings.Repeat("─", fill)) + right
}

func wrapReply(text string, width int) string {
	if width < 8 {
		width = 8
	}
	return ansi.Wrap(text, width, " ")
}

func renderUserMessage(text string, width int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E6EEF8")).
		Background(lipgloss.Color("#243447")).
		Bold(true).
		Padding(0, 1)
	if width > 4 {
		style = style.Width(width)
	}
	return style.Render(text)
}
