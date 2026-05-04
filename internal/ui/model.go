package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/madgodinc/aurora-cli/internal/agent"
	"github.com/madgodinc/aurora-cli/internal/config"
	"github.com/madgodinc/aurora-cli/internal/session"
	"github.com/madgodinc/aurora-cli/internal/tools"
)

type State int

const (
	StateLanding State = iota
	StateChat
)

// SidebarTab controls what sidebar shows
type SidebarTab int

const (
	SidebarTools SidebarTab = iota
	SidebarSessions
)

type Model struct {
	version   string
	state     State
	width     int
	height    int
	input     textarea.Model
	chatView  viewport.Model
	ready     bool
	busy      bool
	streaming bool
	streamBuf string
	startTime time.Time
	cancelCh  chan struct{} // cancel current task

	chatLines []string
	toolLog   []string

	agent        *agent.Agent
	cfg          *config.Config
	inputTokens  int
	outputTokens int
	msgCount     int

	showSidebar  bool
	sidebarWidth int
	sidebarTab   SidebarTab
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

	// Show restored session info
	restored := ag.RestoredMessageCount()
	var chatLines []string
	if restored > 0 {
		chatLines = append(chatLines,
			DimStyle.Render(fmt.Sprintf("  Session restored: %s (%d messages)", ag.SessionID, restored)),
			"",
		)
	}

	return Model{
		version:      version,
		state:        StateLanding,
		agent:        ag,
		cfg:          cfg,
		sidebarWidth: 28,
		showSidebar:  true,
		sidebarTab:   SidebarTools,
		chatLines:    chatLines,
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

const (
	toolPanelH   = 8
	inputHeight  = 4
	headerHeight = 1
)

// ─── UPDATE ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.busy {
				// Cancel current task
				if m.cancelCh != nil {
					close(m.cancelCh)
					m.cancelCh = nil
				}
				m.agent.Cancel()
				m.busy = false
				m.streaming = false
				m.chatLines = append(m.chatLines, ToolErrorStyle.Render("  ✗ Cancelled"), "")
				m.toolLog = append(m.toolLog, ToolErrorStyle.Render("✗ CANCELLED"))
				m.refreshChat()
				return m, nil
			}
			m.agent.SaveSession()
			return m, tea.Quit

		case "esc":
			if m.state == StateLanding {
				return m, tea.Quit
			}
			if m.busy {
				if m.cancelCh != nil {
					close(m.cancelCh)
					m.cancelCh = nil
				}
				m.agent.Cancel()
				m.busy = false
				m.streaming = false
				m.chatLines = append(m.chatLines, ToolErrorStyle.Render("  ✗ Stopped"), "")
				m.refreshChat()
				return m, nil
			}

		case "ctrl+b":
			m.showSidebar = !m.showSidebar
			m.relayout()
			return m, nil

		case "ctrl+t":
			// Toggle sidebar tab
			if m.sidebarTab == SidebarTools {
				m.sidebarTab = SidebarSessions
			} else {
				m.sidebarTab = SidebarTools
			}
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
			m.toolLog = nil
			m.cancelCh = make(chan struct{})

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
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			m.toolLog = append(m.toolLog, ToolStyle.Render("⚡ "+ev.ToolName)+" "+DimStyle.Render(preview))

		case "tool_done":
			result := strings.ReplaceAll(ev.ToolResult, "\n", " ")
			if len(result) > 60 {
				result = result[:60] + "..."
			}
			m.toolLog = append(m.toolLog, ToolDoneStyle.Render("✓ "+ev.ToolName)+" "+DimStyle.Render(result))

		case "error":
			m.toolLog = append(m.toolLog, ToolErrorStyle.Render("✗ "+ev.Text))

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
			m.cancelCh = nil
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
	h := m.height - headerHeight - toolPanelH - inputHeight - 2
	if h < 5 {
		h = 5
	}
	return h
}

func (m *Model) initLayout() {
	mw := m.mainWidth()
	ch := m.chatHeight()
	m.chatView = viewport.New(viewport.WithWidth(mw), viewport.WithHeight(ch))
	ta := textarea.New()
	ta.Placeholder = "Message Aurora... (Esc to cancel)"
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
	m.input.SetWidth(mw - 4)
	m.refreshChat()
}

func (m *Model) updateStreamLine() {
	marker := "♥ Aurora: "
	found := false
	for i := len(m.chatLines) - 1; i >= 0; i-- {
		if strings.Contains(m.chatLines[i], marker) {
			m.chatLines[i] = AuroraNameStyle.Render(marker) + m.streamBuf + DimStyle.Render(" ●")
			found = true
			break
		}
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
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Pink).
		Padding(0, 1).
		MaxWidth(m.mainWidth() - 4)
	for i := len(m.chatLines) - 1; i >= 0; i-- {
		if strings.Contains(m.chatLines[i], marker) {
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

// ─── COMMANDS ───────────────────────────────────────────────────────────────

func (m *Model) handleCommand(input string) string {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.Join(parts[1:], " ")
	}
	switch cmd {
	case "/help":
		return `Commands:
 /new           — new session
 /sessions      — list sessions
 /switch ID     — switch to session
 /compact       — compress history
 /export [file] — export session to markdown
 /import file   — import session from file
 /memory        — show Memory Palace
 /remember k=v  — save to memory
 /clear         — clear display
 /status        — show stats
 /quit          — save & exit
 Ctrl+B sidebar | Ctrl+T tab | Esc cancel`

	case "/quit", "/exit", "/q":
		m.agent.SaveSession()
		// Auto-save session summary to memory
		if m.msgCount > 0 {
			summary := fmt.Sprintf("Session %s in %s, %d messages, %d+%d tokens, %s",
				m.agent.SessionID, filepath.Base(m.agent.WorkDir()),
				m.msgCount, m.inputTokens, m.outputTokens,
				time.Now().Format("2006-01-02 15:04"))
			m.agent.Memory.SaveSummary(summary)
		}
		return "quit"

	case "/new":
		m.agent.NewSession()
		m.chatLines = nil
		m.toolLog = nil
		m.inputTokens = 0
		m.outputTokens = 0
		m.msgCount = 0
		return fmt.Sprintf("New session: %s", m.agent.SessionID)

	case "/sessions":
		sessions := session.List()
		if len(sessions) == 0 {
			return "No sessions."
		}
		var lines []string
		for _, s := range sessions {
			marker := ""
			if s.ID == m.agent.SessionID {
				marker = " ●"
			}
			dir := filepath.Base(s.WorkDir)
			lines = append(lines, fmt.Sprintf(" %s  %s  %d msgs  %s%s",
				s.Updated.Format("01-02 15:04"), s.ID[:15], len(s.Messages), dir, marker))
		}
		return strings.Join(lines, "\n")

	case "/switch":
		if arg == "" {
			return "/switch SESSION_ID"
		}
		// Find session matching prefix
		sessions := session.List()
		for _, s := range sessions {
			if strings.HasPrefix(s.ID, arg) {
				m.agent.SaveSession()
				m.agent.LoadSession(s.ID)
				m.chatLines = []string{DimStyle.Render(fmt.Sprintf("  Switched to session: %s (%d messages)", s.ID, len(s.Messages))), ""}
				m.toolLog = nil
				return ""
			}
		}
		return "Session not found: " + arg

	case "/compact":
		result := m.agent.Compact()
		m.agent.SaveSession()
		return result

	case "/export":
		fname := arg
		if fname == "" {
			fname = fmt.Sprintf("aurora_session_%s.md", time.Now().Format("20060102_150405"))
		}
		md := m.agent.ExportMarkdown()
		if err := os.WriteFile(fname, []byte(md), 0644); err != nil {
			return "Export error: " + err.Error()
		}
		return "Exported to: " + fname

	case "/import":
		if arg == "" {
			return "/import filename.json"
		}
		data, err := os.ReadFile(arg)
		if err != nil {
			return "Read error: " + err.Error()
		}
		var s session.Session
		if err := json.Unmarshal(data, &s); err != nil {
			return "Parse error: " + err.Error()
		}
		session.Save(&s)
		return fmt.Sprintf("Imported session %s (%d messages)", s.ID, len(s.Messages))

	case "/clear":
		m.chatLines = nil
		m.toolLog = nil
		return ""

	case "/status":
		return fmt.Sprintf("Session: %s\nMsgs: %d | Tokens: %d↑ %d↓ | Memory: %d | Tools: %d\nDir: %s",
			m.agent.SessionID, m.msgCount, m.inputTokens, m.outputTokens,
			m.agent.Memory.Count(), tools.ToolCount(m.cfg.HasSSH), m.agent.WorkDir())

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

	case "/cd":
		if arg == "" {
			return "Dir: " + m.agent.WorkDir()
		}
		info, err := os.Stat(arg)
		if err != nil || !info.IsDir() {
			return "Not a directory: " + arg
		}
		abs, _ := filepath.Abs(arg)
		os.Chdir(abs)
		m.agent.SetWorkDir(abs)
		return "Dir: " + abs

	default:
		return "Unknown: " + cmd + " (/help)"
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
	restored := m.agent.RestoredMessageCount()
	info := DimStyle.Render(fmt.Sprintf("   Model: Qwen3.6-27B  |  Tools: %d  |  User: %s", tc, m.cfg.Username))
	sessionInfo := ""
	if restored > 0 {
		sessionInfo = DimStyle.Render(fmt.Sprintf("   Session: %s (%d messages)", m.agent.SessionID[:15], restored))
	}
	enter := DimStyle.Render("\n   Press ENTER to start  ·  ESC to quit")
	parts := []string{b, "", ver, "   " + status, info}
	if sessionInfo != "" {
		parts = append(parts, sessionInfo)
	}
	parts = append(parts, enter)
	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) viewChat() string {
	mw := m.mainWidth()

	// ── Top: Logo + Tool Activity ──
	logoW := 16
	toolPanelW := mw - logoW - 1

	logo := lipgloss.NewStyle().
		Width(logoW).
		Foreground(Pink).Bold(true).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PinkMuted).
		Render("♥ AURORA")

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
			toolContent = DimStyle.Render("  ready")
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
	logoStyled := lipgloss.NewStyle().Height(toolPanelH).Render(logo)
	topBar := lipgloss.JoinHorizontal(lipgloss.Top, logoStyled, " ", toolPanel)

	// ── Status line ──
	busyStr := ""
	if m.busy {
		busyStr = " ⟳ Esc=cancel"
	}
	statusText := fmt.Sprintf(" ♥ v%s │ %s │ %d↑ %d↓%s │ Mem:%d │ Ctrl+B Ctrl+T ",
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

	mainCol := lipgloss.JoinVertical(lipgloss.Left, topBar, statusLine, chatContent, sep, inputLine)

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
	sep := DimStyle.Render(strings.Repeat("─", w))

	switch m.sidebarTab {
	case SidebarSessions:
		return m.renderSessionsSidebar(w, sep)
	default:
		return m.renderToolsSidebar(w, sep)
	}
}

func (m Model) renderToolsSidebar(w int, sep string) string {
	title := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Tools [Ctrl+T]")

	var toolLines []string
	for _, t := range tools.Registry(m.cfg.HasSSH) {
		toolLines = append(toolLines, DimStyle.Render(" "+t.Def.Name))
	}

	memTitle := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Memory")
	memCount := DimStyle.Render(fmt.Sprintf(" %d items", m.agent.Memory.Count()))

	infoTitle := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Info")
	info := []string{
		DimStyle.Render(" " + m.cfg.Username),
		DimStyle.Render(" Qwen3.6-27B"),
		DimStyle.Render(fmt.Sprintf(" %d msgs", m.msgCount)),
		DimStyle.Render(" " + m.agent.SessionID[:min(15, len(m.agent.SessionID))]),
	}
	if m.cfg.HasSSH {
		info = append(info, lipgloss.NewStyle().Foreground(Green).Render(" ● "+m.cfg.SSHAlias))
	}

	parts := []string{"", title, ""}
	parts = append(parts, toolLines...)
	parts = append(parts, "", sep, "", memTitle, memCount, "", sep, "", infoTitle)
	parts = append(parts, info...)

	return lipgloss.NewStyle().Width(m.sidebarWidth).Height(m.height - 1).
		Render(strings.Join(parts, "\n"))
}

func (m Model) renderSessionsSidebar(w int, sep string) string {
	title := lipgloss.NewStyle().Foreground(Pink).Bold(true).
		Width(w).Align(lipgloss.Center).Render("♥ Sessions [Ctrl+T]")

	sessions := session.List()
	var lines []string
	for _, s := range sessions {
		marker := " "
		if s.ID == m.agent.SessionID {
			marker = "●"
		}
		id := s.ID
		if len(id) > 12 {
			id = id[:12]
		}
		dir := filepath.Base(s.WorkDir)
		if len(dir) > 10 {
			dir = dir[:10]
		}
		line := fmt.Sprintf("%s %s %dm %s", marker, id, len(s.Messages), dir)
		if s.ID == m.agent.SessionID {
			lines = append(lines, lipgloss.NewStyle().Foreground(Pink).Render(line))
		} else {
			lines = append(lines, DimStyle.Render(line))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, DimStyle.Render(" no sessions"))
	}

	helpLines := []string{
		"",
		sep,
		DimStyle.Render(" /new      new"),
		DimStyle.Render(" /switch   load"),
		DimStyle.Render(" /export   save md"),
		DimStyle.Render(" /import   load"),
		DimStyle.Render(" /compact  compress"),
	}

	parts := []string{"", title, ""}
	parts = append(parts, lines...)
	parts = append(parts, helpLines...)

	return lipgloss.NewStyle().Width(m.sidebarWidth).Height(m.height - 1).
		Render(strings.Join(parts, "\n"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
