package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kocoro/edu-cli/internal/session"
)

const (
	stateStartup = iota
	stateInput
	stateProcessing
	stateSessionPicker
)

const inputBorderOverhead = 3

// ModelChoice is one /model row. Detail is optional, such as an Ollama size.
type ModelChoice struct {
	Name   string
	Detail string
}

// Options is the home screen plus the runner callback.
type Options struct {
	Agent     string
	Model     string
	Endpoint  string
	CWD       string
	Version   string
	Connected bool
	Activity  []Activity
	// Sessions is the Kocoro-style session harness (picker, resume, save).
	Sessions *session.Manager
	// Submit runs one turn and returns the assistant reply.
	Submit func(text string) (string, error)
	// Models lists DeepSeek ids plus any local Ollama tags. Nil means DeepSeek only.
	Models func() []ModelChoice
	// ApplyModel retargets later turns. It returns the endpoint now in use.
	ApplyModel func(name string) (endpoint string, err error)
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
	history           string
	baseHeader        string
	spin              int
	started           time.Time
	lastUserInput     string
	lastSessions      []session.SessionSummary
	sessionPickerIdx  int
}

// Start opens the full-screen CLI. The line reader in Run stays for pipes.
func Start(opt Options) error {
	if opt.Version == "" {
		opt.Version = "dev"
	}
	if opt.Agent == "" {
		opt.Agent = "default"
	}
	p := tea.NewProgram(newModel(opt), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func newModel(opt Options) *Model {
	width := 80
	ta := textarea.New()
	ta.Placeholder = "Ask Edu anything…"
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
		m.persistSessionAfterTurn()
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
				m.saveSessions()
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
		if m.state == stateSessionPicker {
			switch msg.Type {
			case tea.KeyUp:
				m.sessionPickerIdx--
				if m.sessionPickerIdx < 0 {
					m.sessionPickerIdx = len(m.lastSessions) - 1
				}
				return m, nil
			case tea.KeyDown:
				m.sessionPickerIdx++
				if m.sessionPickerIdx >= len(m.lastSessions) {
					m.sessionPickerIdx = 0
				}
				return m, nil
			case tea.KeyEnter:
				if len(m.lastSessions) > 0 {
					id := m.lastSessions[m.sessionPickerIdx].ID
					if sess, err := m.opt.Sessions.Resume(id); err != nil {
						m.append(styleError().Render("  "+err.Error()) + "\n")
					} else {
						m.loadSessionHistory(sess)
					}
				}
				m.state = stateInput
				m.relayout()
				return m, textarea.Blink
			case tea.KeyEsc:
				m.state = stateInput
				m.relayout()
				return m, nil
			}
			return m, nil
		}
		if m.state != stateStartup && m.scrollTranscript(msg) {
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
				m.saveSessions()
				return m, tea.Quit
			}
			if strings.HasPrefix(text, "/") {
				if text == "/quit" || text == "/exit" {
					m.saveSessions()
					return m, tea.Quit
				}
				if m.handleSlashCommand(text) {
					return m, nil
				}
			}
			if text == "?" || text == "/help" || text == "help" {
				m.help = true
				m.relayout()
				return m, nil
			}
			if strings.HasPrefix(text, "/model") {
				m.help = false
				m.append(renderUserMessage(text, m.width))
				m.append(m.modelCommand(text))
				m.append("")
				return m, nil
			}
			m.help = false
			m.lastUserInput = text
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
	case tea.MouseMsg:
		if m.state == stateStartup {
			return m, nil
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	if m.state == stateInput {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

// scrollTranscript moves the conversation pane. Up and Down do this only
// when the composer is empty, so typed lines stay in the input box.
func (m *Model) scrollTranscript(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyPgUp:
		m.viewport.PageUp()
	case tea.KeyPgDown:
		m.viewport.PageDown()
	case tea.KeyHome:
		m.viewport.GotoTop()
	case tea.KeyEnd:
		m.viewport.GotoBottom()
	case tea.KeyUp:
		if m.state == stateProcessing || strings.TrimSpace(m.textarea.Value()) == "" {
			m.viewport.LineUp(1)
		} else {
			return false
		}
	case tea.KeyDown:
		if m.state == stateProcessing || strings.TrimSpace(m.textarea.Value()) == "" {
			m.viewport.LineDown(1)
		} else {
			return false
		}
	default:
		return false
	}
	return true
}

func (m *Model) modelCommand(text string) string {
	fields := strings.Fields(text)
	if len(fields) >= 2 {
		name := strings.Join(fields[1:], " ")
		if m.opt.ApplyModel == nil {
			return "  /model is not wired to a client"
		}
		endpoint, err := m.opt.ApplyModel(name)
		if err != nil {
			return "  " + err.Error()
		}
		m.opt.Model = strings.TrimPrefix(strings.TrimPrefix(name, "ollama/"), "deepseek/")
		if endpoint != "" {
			m.opt.Endpoint = endpoint
		}
		return "  Switched to " + m.modelLabel()
	}
	return formatModelList(m.modelChoices(), m.opt.Model)
}

func (m *Model) modelChoices() []ModelChoice {
	if m.opt.Models != nil {
		return m.opt.Models()
	}
	return []ModelChoice{
		{Name: "deepseek-chat"},
		{Name: "deepseek-reasoner"},
		{Name: "deepseek-flash"},
	}
}

func formatModelList(choices []ModelChoice, current string) string {
	current = strings.TrimPrefix(strings.TrimPrefix(current, "ollama/"), "deepseek/")
	var b strings.Builder
	b.WriteString("Available models:\n")
	for _, c := range choices {
		mark := "  "
		if c.Name == current {
			mark = "→ "
		}
		line := mark + c.Name
		if c.Detail != "" {
			line += " (" + c.Detail + ")"
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("Use /model <name> to switch")
	return b.String()
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
	m.baseHeader = m.history
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
	out := m.viewport.View()
	if m.state == stateSessionPicker {
		var sb strings.Builder
		sb.WriteString(styleDim().Render("  Sessions (↑↓, Enter, Esc)"))
		sb.WriteString("\n")
		sb.WriteString(renderDropList(dropListSize, len(m.lastSessions), m.sessionPickerIdx, func(i int) (string, string) {
			s := m.lastSessions[i]
			title := s.Title
			if len(title) > 28 {
				title = title[:25] + "..."
			}
			when := s.UpdatedAt
			if when.IsZero() {
				when = s.CreatedAt
			}
			desc := fmt.Sprintf("%s · %d msgs", when.Format("Jan 02 15:04"), s.MsgCount)
			return title, desc
		}))
		out += "\n" + sb.String()
	}
	return out + "\n" + m.bottomRegion()
}

func (m *Model) bottomRegion() string {
	var sb strings.Builder
	if m.help && m.state == stateInput {
		sb.WriteString(styleDim().Render("  ? commands  /sessions resume  PgUp/PgDn scroll  exit quit  Ctrl+C twice"))
		sb.WriteString("\n")
	}
	switch m.state {
	case stateProcessing:
		sb.WriteString(renderInputBox(styleDim().Render("> "), m.width))
	case stateSessionPicker:
		sb.WriteString(styleDim().Render("  ↑↓ choose · Enter resume · Esc cancel"))
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
	if strings.Contains(ep, "deepseek") && !strings.Contains(model, "/") {
		return "deepseek/" + model
	}
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

func (m *Model) saveSessions() {
	if m.opt.Sessions != nil {
		_ = m.opt.Sessions.Save()
	}
}

func (m *Model) persistSessionAfterTurn() {
	if m.opt.Sessions == nil {
		return
	}
	_ = m.opt.Sessions.RefreshCurrent()
	cur := m.opt.Sessions.Current()
	if cur != nil && cur.Title == "New session" && m.lastUserInput != "" {
		if t := session.TitleFromUserLine(m.lastUserInput); t != "" {
			cur.Title = t
		}
	}
	_ = m.opt.Sessions.Save()
}

func (m *Model) handleSlashCommand(input string) bool {
	if m.opt.Sessions == nil {
		return false
	}
	parts := strings.Fields(input)
	cmd := parts[0]
	switch cmd {
	case "/clear":
		m.history = m.baseHeader
		m.viewport.SetContent(m.history)
		m.viewport.GotoBottom()
		return true
	case "/sessions":
		list, err := m.opt.Sessions.List()
		if err != nil {
			m.append(styleError().Render("  "+err.Error()) + "\n")
			return true
		}
		if len(list) == 0 {
			m.append("  No saved sessions\n")
			return true
		}
		m.lastSessions = list
		m.sessionPickerIdx = 0
		m.state = stateSessionPicker
		m.relayout()
		return true
	case "/session":
		if len(parts) < 2 {
			m.append("  Usage: /session new | /session resume <# or id>\n")
			return true
		}
		switch parts[1] {
		case "new":
			if _, err := m.opt.Sessions.NewSession(); err != nil {
				m.append(styleError().Render("  "+err.Error()) + "\n")
				return true
			}
			m.history = m.baseHeader
			m.append("  Started new session\n")
			m.viewport.SetContent(m.history)
			m.viewport.GotoBottom()
			return true
		case "resume":
			if len(parts) < 3 {
				m.append("  Usage: /session resume <number or id>\n")
				return true
			}
			target := parts[2]
			var sess *session.Session
			var err error
			if n, perr := strconv.Atoi(target); perr == nil {
				sess, err = m.opt.Sessions.ResumeByListIndex(n)
			} else {
				sess, err = m.opt.Sessions.Resume(target)
			}
			if err != nil {
				m.append(styleError().Render("  "+err.Error()) + "\n")
				return true
			}
			m.loadSessionHistory(sess)
			return true
		}
	}
	return false
}

func (m *Model) loadSessionHistory(sess *session.Session) {
	if sess == nil {
		return
	}
	m.history = m.baseHeader
	m.append(styleDim().Render(fmt.Sprintf("  Session: %s", session.DisplayTitle(sess))))
	m.append("")
	for _, msg := range session.MessagesForDisplay(sess.Messages) {
		switch msg.Role {
		case "user":
			m.append(renderUserMessage(msg.Content, m.width))
		case "assistant":
			m.append(wrapReply(msg.Content, m.width))
			m.append("")
		}
	}
	m.viewport.SetContent(m.history)
	m.viewport.GotoBottom()
}

func renderUserMessage(text string, width int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ECFDF5")).
		Background(lipgloss.Color("#14532D")).
		Bold(true).
		Padding(0, 1)
	if width > 4 {
		style = style.Width(width)
	}
	return style.Render(text)
}
