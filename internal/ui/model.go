package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/madgodinc/aurora-cli/internal/agent"
	"github.com/madgodinc/aurora-cli/internal/config"
	"github.com/madgodinc/aurora-cli/internal/tools"
)

type State int

const (
	StateLanding State = iota
	StateChat
)

type Model struct {
	version   string
	state     State
	width     int
	height    int
	input     textarea.Model
	chatView  viewport.Model
	toolView  viewport.Model
	ready     bool
	busy      bool
	streaming bool
	streamBuf string
	startTime time.Time

	// Chat entries (text only)
	chatLines []string
	// Tool activity log
	toolLog []string

	agent        *agent.Agent
	cfg          *config.Config
	inputTokens  int
	outputTokens int
	msgCount     int
	showSidebar  bool
	sidebarWidth int
}

type agentEventMsg agent.Event

func NewModel(version string, cfg *config.Config) Model {
	wd, _ := os.Getwd()
	agCfg := agent.Config{
		ProxyURL: cfg.ProxyURL,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
		WorkDir:  wd,
		Username: cfg.Username,
		HasSSH:   cfg.HasSSH,
		Token:    cfg.Token,
		SSHAlias: cfg.SSHAlias,
	}
	ag := agent.New(agCfg)
	return Model{
		version:      version,
		state:        StateLanding,
		agent:        ag,
		cfg:          cfg,
		sidebarWidth: 24,
		showSidebar:  true,
	}
}

func (m Model) Init() tea.Cmd { return textarea.Blink }

func listenAgent(ch <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return agentEventMsg(agent.Event{Type: "done"})
		}
		return agentEventMsg(ev)
	}
}

// ─── Layout constants ───
const (
	logoHeight    = 4
	toolPanelH    = 10  // tool activity panel height
	inputHeight   = 4
	headerHeight  = 1   // status bar
)

// ─── UPDATE ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state == StateLanding {
				return m, tea.Quit
			}
		case "ctrl+b":
			m.showSidebar = !m.showSidebar
			m.relayout()
			return m, nil
		case "enter":
			if m.state == StateLanding {
				m.state = StateChat
				return m, nil
			}
			if m.busy {
				return m, nil
			}
			val := m.input.Value()
			if strings.TrimSpace(val) == "" {
				return m, nil
			}
			if strings.HasPrefix(val, "/") {
				result := m.handleCommand(val)
				if result == "quit" {
					return m, tea.Quit
				}
				if result != "" {
					m.chatLines = append(m.chatLines, DimStyle.Render(result), "")
					m.refreshChat()
				}
				m.input.Reset()
				return m, nil
			}

			m.chatLines = append(m.chatLines, PromptStyle.Render("♥ You: ")+val, "")
			m.input.Reset()
			m.busy = true
			m.streaming = false
			m.streamBuf = ""
			m.startTime = time.Now()
			m.msgCount++
			m.toolLog = nil // clear tool log for new turn

			ch := m.agent.Events()
			go m.agent.Run(val)
			m.refreshChat()
			return m, listenAgent(ch)
		}

	case agentEventMsg:
		ev := agent.Event(msg)
		switch ev.Type {
		case "text":
			m.streamBuf += ev.Text
			m.streaming = true
			m.updateStreamLine()

		case "tool_start":
			if m.streaming {
				m.finalizeStream()
			}
			preview := ev.ToolInput
			if preview == "" {
				preview = ""
			}
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			m.toolLog = append(m.toolLog, ToolStyle.Render("⚡ "+ev.ToolName)+" "+DimStyle.Render(preview))
			m.refreshToolView()

		case "tool_done":
			result := ev.ToolResult
			if len(result) > 60 {
				result = result[:60] + "..."
			}
			result = strings.ReplaceAll(result, "\n", " ")
			m.toolLog = append(m.toolLog, ToolDoneStyle.Render("✓ "+ev.ToolName)+" "+DimStyle.Render(result))
			m.refreshToolView()

		case "error":
			m.toolLog = append(m.toolLog, ToolErrorStyle.Render("✗ "+ev.Text))
			m.refreshToolView()

		case "tokens":
			m.inputTokens += ev.InputTokens
			m.outputTokens += ev.OutputTokens

		case "done":
			if m.streaming {
				m.finalizeStream()
			}
			elapsed := time.Since(m.startTime).Seconds()
			if elapsed > 0.5 {
				m.chatLines = append(m.chatLines, DimStyle.Render(fmt.Sprintf("  %.1fs", elapsed)), "")
			}
			m.busy = false
			m.refreshChat()
			return m, nil
		}

		m.refreshChat()
		ch := m.agent.Events()
		return m, listenAgent(ch)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.initLayout()
			m.ready = true
		} else {
			m.relayout()
		}
		return m, nil
	}

	if m.state == StateChat && !m.busy {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == StateChat {
		var cmd tea.Cmd
		m.chatView, cmd = m.chatView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) mainWidth() int {
	if m.showSidebar {
		return m.width - m.sidebarWidth - 1
	}
	return m.width
}

func (m *Model) chatHeight() int {
	h := m.height - headerHeight - toolPanelH - inputHeight - 2 // borders/separators
	if h < 5 {
		h = 5
	}
	return h
}

func (m *Model) initLayout() {
	mw := m.mainWidth()
	ch := m.chatHeight()

	m.chatView = viewport.New(viewport.WithWidth(mw), viewport.WithHeight(ch))
	m.toolView = viewport.New(viewport.WithWidth(mw/2), viewport.WithHeight(toolPanelH-2))

	ta := textarea.New()
	ta.Placeholder = "Message Aurora..."
	ta.Focus()
	ta.CharLimit = 10000
	ta.SetWidth(mw - 4)
	ta.SetHeight(2)
	ta.ShowLineNumbers = false
	m.input = ta
}

func (m *Model) relayout() {
	mw := m.mainWidth()
	ch := m.chatHeight()
	m.chatView.SetWidth(mw)
	m.chatView.SetHeight(ch)
	m.toolView.SetWidth(mw / 2)
	m.toolView.SetHeight(toolPanelH - 2)
	m.input.SetWidth(mw - 4)
	m.refreshChat()
	m.refreshToolView()
}

func (m *Model) updateStreamLine() {
	// Find or append streaming line
	marker := "♥ Aurora: "
	found := false
	for i := len(m.chatLines) - 1; i >= 0; i-- {
		if strings.Contains(m.chatLines[i], marker) || strings.Contains(m.chatLines[i], "aurora_stream_marker") {
			m.chatLines[i] = AuroraNameStyle.Render(marker) + m.streamBuf
			if m.busy {
				m.chatLines[i] += DimStyle.Render(" ●")
			}
			found = true
			break
		}
		// Stop looking past user messages
		if strings.Contains(m.chatLines[i], "♥ You:") {
			break
		}
	}
	if !found {
		m.chatLines = append(m.chatLines, AuroraNameStyle.Render(marker)+m.streamBuf+DimStyle.Render(" ●"))
	}
}

func (m *Model) finalizeStream() {
	if m.streamBuf == "" {
		return
	}
	marker := "♥ Aurora: "
	rendered := renderMarkdown(m.streamBuf)
	for i := len(m.chatLines) - 1; i >= 0; i-- {
		if strings.Contains(m.chatLines[i], marker) {
			// Replace with rendered markdown in a border box
			border := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Pink).
				Padding(0, 1).
				MaxWidth(m.mainWidth() - 4)
			m.chatLines[i] = AuroraNameStyle.Render("♥ Aurora") + "\n" + border.Render(rendered)
			break
		}
	}
	m.chatLines = append(m.chatLines, "")
	m.streamBuf = ""
	m.streaming = false
}

func (m *Model) refreshChat() {
	m.chatView.SetContent(strings.Join(m.chatLines, "\n"))
	m.chatView.GotoBottom()
}

func (m *Model) refreshToolView() {
	// Show last N tool log entries that fit
	maxLines := toolPanelH - 2
	start := 0
	if len(m.toolLog) > maxLines {
		start = len(m.toolLog) - maxLines
	}
	m.toolView.SetContent(strings.Join(m.toolLog[start:], "\n"))
	m.toolView.GotoBottom()
}

func (m *Model) handleCommand(input string) string {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.Join(parts[1:], " ")
	}
	switch cmd {
	case "/help":
		return `Commands: /help /compact /memory /remember /clear /status /quit (Ctrl+B sidebar)`
	case "/quit", "/exit", "/q":
		return "quit"
	case "/compact":
		return m.agent.Compact()
	case "/clear":
		m.chatLines = nil
		m.toolLog = nil
		return ""
	case "/status":
		return fmt.Sprintf("Msgs: %d | Tokens: %d↑ %d↓ | Memory: %d | Tools: %d",
			m.msgCount, m.inputTokens, m.outputTokens, m.agent.Memory.Count(), tools.ToolCount(m.cfg.HasSSH))
	case "/memory":
		mem := m.agent.Memory.RecallAll()
		if mem == "" {
			return "Memory Palace is empty."
		}
		return mem
	case "/remember":
		if arg == "" {
			return "/remember key = value"
		}
		if strings.Contains(arg, "=") {
			kv := strings.SplitN(arg, "=", 2)
			m.agent.Memory.Remember("user", strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
			return "Saved."
		}
		m.agent.Memory.AddFact(arg)
		return "Fact saved."
	default:
		return "Unknown: " + cmd
	}
}

// ─── VIEW ───────────────────────────────────────────────────────────────────

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	if !m.ready {
		v.Content = "Loading..."
		return v
	}
	switch m.state {
	case StateLanding:
		v.Content = m.viewLanding()
	case StateChat:
		v.Content = m.viewChat()
	}
	return v
}

func (m Model) viewLanding() string {
	b := BannerStyle.Render(banner)
	ver := SubtitleStyle.Render(fmt.Sprintf("v%s", m.version))
	status := lipgloss.NewStyle().Foreground(Green).Render("● brain::online")
	tc := tools.ToolCount(m.cfg.HasSSH)
	info := DimStyle.Render(fmt.Sprintf("   Model: Qwen3.6-27B  |  Tools: %d  |  User: %s", tc, m.cfg.Username))
	enter := DimStyle.Render("\n   Press ENTER to start  ·  ESC to quit")
	content := lipgloss.JoinVertical(lipgloss.Left, b, "", ver, "   "+status, info, enter)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) viewChat() string {
	mw := m.mainWidth()

	// ── Top bar: Logo + Tool Activity ──
	logoW := 16
	toolPanelW := mw - logoW - 1

	logo := lipgloss.NewStyle().
		Width(logoW).
		Foreground(Pink).Bold(true).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PinkMuted).
		Render("♥ AURORA")

	// Tool activity panel
	toolBorder := lipgloss.NewStyle().
		Width(toolPanelW).Height(toolPanelH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Yellow).
		Padding(0, 1)

	toolTitle := ToolStyle.Render("⚡ Activity")
	var toolContent string
	if len(m.toolLog) == 0 {
		if m.busy {
			toolContent = DimStyle.Render("  thinking...")
		} else {
			toolContent = DimStyle.Render("  waiting for task...")
		}
	} else {
		maxShow := toolPanelH - 3
		start := 0
		if len(m.toolLog) > maxShow {
			start = len(m.toolLog) - maxShow
		}
		toolContent = strings.Join(m.toolLog[start:], "\n")
	}
	toolPanel := toolBorder.Render(toolTitle + "\n" + toolContent)

	// Make logo same height as tool panel
	logoStyled := lipgloss.NewStyle().Height(toolPanelH).Render(logo)

	topBar := lipgloss.JoinHorizontal(lipgloss.Top, logoStyled, " ", toolPanel)

	// ── Status line ──
	busyStr := ""
	if m.busy {
		busyStr = " ⟳"
	}
	statusText := fmt.Sprintf(" ♥ v%s │ %s │ %d↑ %d↓%s │ Mem: %d │ Ctrl+B ",
		m.version, m.cfg.Username, m.inputTokens, m.outputTokens, busyStr, m.agent.Memory.Count())
	statusLine := StatusBarStyle.Width(mw).Render(statusText)

	// ── Chat ──
	chatContent := m.chatView.View()

	// ── Input ──
	sep := SeparatorStyle.Render(strings.Repeat("─", mw))
	promptIcon := PromptStyle.Render("♥ ")
	if m.busy {
		promptIcon = DimStyle.Render("⟳ ")
	}
	inputLine := promptIcon + m.input.View()

	// ── Main column ──
	mainCol := lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		statusLine,
		chatContent,
		sep,
		inputLine,
	)

	// ── Sidebar ──
	if m.showSidebar {
		sidebar := m.renderSidebar()
		sepV := lipgloss.NewStyle().Foreground(PinkMuted).
			Render(strings.Repeat("│\n", m.height-1))
		return lipgloss.JoinHorizontal(lipgloss.Top, mainCol, sepV, sidebar)
	}

	return mainCol
}

func (m Model) renderSidebar() string {
	w := m.sidebarWidth - 2

	title := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Tools")

	var toolLines []string
	for _, t := range tools.Registry(m.cfg.HasSSH) {
		toolLines = append(toolLines, DimStyle.Render(" "+t.Def.Name))
	}

	sep := DimStyle.Render(strings.Repeat("─", w))

	memTitle := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Memory")
	memCount := DimStyle.Render(fmt.Sprintf(" %d items", m.agent.Memory.Count()))

	infoTitle := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Info")
	info := []string{
		DimStyle.Render(" " + m.cfg.Username),
		DimStyle.Render(" Qwen3.6-27B"),
		DimStyle.Render(fmt.Sprintf(" %d msgs", m.msgCount)),
	}
	if m.cfg.HasSSH {
		info = append(info, lipgloss.NewStyle().Foreground(Green).Render(" ● "+m.cfg.SSHAlias))
	}

	parts := []string{"", title, ""}
	parts = append(parts, toolLines...)
	parts = append(parts, "", sep, "", memTitle, memCount)
	parts = append(parts, "", sep, "", infoTitle)
	parts = append(parts, info...)

	return lipgloss.NewStyle().Width(m.sidebarWidth).Height(m.height - 1).
		Render(strings.Join(parts, "\n"))
}
