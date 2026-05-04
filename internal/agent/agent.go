package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/madgodinc/aurora-cli/internal/memory"
	"github.com/madgodinc/aurora-cli/internal/provider"
	"github.com/madgodinc/aurora-cli/internal/session"
	"github.com/madgodinc/aurora-cli/internal/tools"
)

const (
	MaxToolRounds = 30
	MaxTokens     = 8192
)

// Config for the agent.
type Config struct {
	ProxyURL string
	APIKey   string
	Model    string
	WorkDir  string
	Username string
	HasSSH   bool
	Token    string
	SSHAlias string
}

// Event is emitted by the agent during processing.
type Event struct {
	Type string // "text", "tool_start", "tool_done", "error", "done", "tokens"

	// Text content
	Text string

	// Tool info
	ToolName  string
	ToolInput string // preview
	ToolResult string

	// Token usage
	InputTokens  int
	OutputTokens int
}

// Agent manages the conversation and tool loop.
type Agent struct {
	config    Config
	client    *provider.Client
	messages  []provider.Message
	toolDefs  []provider.ToolDef
	eventCh   chan Event
	Memory    *memory.Palace
	SessionID string
	cancelled bool
}

// New creates a new agent.
func New(cfg Config) *Agent {
	client := provider.NewClient(provider.Config{
		BaseURL: cfg.ProxyURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	})

	ag := &Agent{
		config:   cfg,
		client:   client,
		messages: make([]provider.Message, 0),
		toolDefs: tools.ToolDefs(cfg.HasSSH),
		Memory:   memory.New(cfg.Token),
	}

	// Set SSH host for remote tools
	if cfg.SSHAlias != "" {
		tools.SSHHost = cfg.SSHAlias
	}

	// Connect Remember tool to Memory Palace
	tools.MemoryCallback = func(action, category, key, value string) string {
		switch action {
		case "save":
			if category == "fact" {
				ag.Memory.AddFact(key + ": " + value)
				return fmt.Sprintf("Remembered fact: %s = %s", key, value)
			}
			ag.Memory.Remember(category, key, value)
			return fmt.Sprintf("Remembered [%s] %s = %s", category, key, value)
		case "recall":
			mem := ag.Memory.RecallAll()
			if mem == "" {
				return "Memory Palace is empty."
			}
			return mem
		case "list":
			return fmt.Sprintf("Memory: %d items\n%s", ag.Memory.Count(), ag.Memory.RecallAll())
		}
		return "Unknown action"
	}

	// Try to restore last session for this directory
	ag.restoreSession()

	return ag
}

// Events returns the event channel. Must be called before Run.
func (a *Agent) Events() <-chan Event {
	if a.eventCh == nil {
		a.eventCh = make(chan Event, 100)
	}
	return a.eventCh
}

func (a *Agent) emit(e Event) {
	if a.eventCh != nil {
		a.eventCh <- e
	}
}

// MessageCount returns the number of messages.
func (a *Agent) MessageCount() int {
	return len(a.messages)
}

// Run processes a user message through the full agent loop.
// This should be called in a goroutine. Events are emitted via the channel.
func (a *Agent) Run(userInput string) {
	if a.eventCh == nil {
		a.eventCh = make(chan Event, 100)
	}
	a.cancelled = false
	defer func() {
		a.SaveSession()
		a.emit(Event{Type: "done"})
	}()

	// Auto-compact if too many messages
	if len(a.messages) > 50 {
		a.Compact()
	}

	// Add user message
	a.messages = append(a.messages, provider.Message{
		Role:    "user",
		Content: userInput,
	})

	system := buildSystemPrompt(a.config, a.Memory)

	for round := 0; round < MaxToolRounds; round++ {
		if a.cancelled {
			return
		}
		// Build request
		req := provider.Request{
			Model:     a.config.Model,
			MaxTokens: MaxTokens,
			System:    system,
			Messages:  a.messages,
			Tools:     a.toolDefs,
		}

		// Stream response
		events, err := a.client.Stream(req)
		if err != nil {
			a.emit(Event{Type: "error", Text: err.Error()})
			return
		}

		// Collect response
		var textBuf strings.Builder
		var toolCalls []toolCall
		var currentTool toolCall
		var stopReason string
		inTool := false

		for ev := range events {
			switch ev.Type {
			case "message_start":
				a.emit(Event{Type: "tokens", InputTokens: ev.InputTokens})

			case "text":
				textBuf.WriteString(ev.Text)
				a.emit(Event{Type: "text", Text: ev.Text})

			case "tool_use_start":
				inTool = true
				currentTool = toolCall{
					ID:   ev.ToolID,
					Name: ev.ToolName,
				}
				a.emit(Event{Type: "tool_start", ToolName: ev.ToolName})

			case "tool_input_delta":
				currentTool.InputJSON += ev.ToolJSON

			case "block_stop":
				if inTool {
					toolCalls = append(toolCalls, currentTool)
					inTool = false
				}

			case "message_delta":
				stopReason = ev.StopReason
				a.emit(Event{Type: "tokens", OutputTokens: ev.OutputTokens})
			}
		}

		// Build assistant content for history
		var assistantContent []provider.ContentBlock
		if textBuf.Len() > 0 {
			assistantContent = append(assistantContent, provider.ContentBlock{
				Type: "text",
				Text: textBuf.String(),
			})
		}
		for _, tc := range toolCalls {
			var input interface{}
			json.Unmarshal([]byte(tc.InputJSON), &input)
			assistantContent = append(assistantContent, provider.ContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Name,
				Input: input,
			})
		}
		if len(assistantContent) > 0 {
			a.messages = append(a.messages, provider.Message{
				Role:    "assistant",
				Content: assistantContent,
			})
		}

		// No tools → done
		if len(toolCalls) == 0 {
			break
		}

		// Execute tools
		var toolResults []provider.ContentBlock
		for _, tc := range toolCalls {
			var input map[string]interface{}
			json.Unmarshal([]byte(tc.InputJSON), &input)
			if input == nil {
				input = make(map[string]interface{})
			}

			// Preview
			preview := toolPreview(tc.Name, input)
			a.emit(Event{Type: "tool_start", ToolName: tc.Name, ToolInput: preview})

			// Execute
			t := tools.FindTool(tc.Name, a.config.HasSSH)
			var result string
			if t != nil {
				result = t.Execute(input)
			} else {
				result = fmt.Sprintf("Unknown tool: %s", tc.Name)
			}

			// Truncate
			if len(result) > 30000 {
				result = result[:30000] + "\n... (truncated)"
			}

			toolResults = append(toolResults, provider.ContentBlock{
				Type:      "tool_result",
				ToolUseID: tc.ID,
				Content:   result,
			})

			resultPreview := result
			if len(resultPreview) > 100 {
				resultPreview = resultPreview[:100] + "..."
			}
			a.emit(Event{Type: "tool_done", ToolName: tc.Name, ToolResult: resultPreview})
		}

		// Add tool results as user message
		a.messages = append(a.messages, provider.Message{
			Role:    "user",
			Content: toolResults,
		})

		// If stop reason was end_turn, stop after tool execution
		if stopReason == "end_turn" {
			break
		}
	}
}

// Compact compresses conversation history.
func (a *Agent) Compact() string {
	if len(a.messages) <= 10 {
		return "Too few messages to compact."
	}
	keep := 8
	old := a.messages[:len(a.messages)-keep]
	var summary strings.Builder
	for _, m := range old {
		role := m.Role
		switch c := m.Content.(type) {
		case string:
			if len(c) > 200 {
				c = c[:200]
			}
			fmt.Fprintf(&summary, "%s: %s\n", role, c)
		}
	}
	a.messages = append(
		[]provider.Message{
			{Role: "user", Content: fmt.Sprintf("[Compacted %d messages]\n%s", len(old), summary.String())},
			{Role: "assistant", Content: "I have the context. Continuing."},
		},
		a.messages[len(a.messages)-keep:]...,
	)
	return fmt.Sprintf("Compacted %d messages.", len(old))
}

func (a *Agent) restoreSession() {
	s := session.FindLatest(a.config.WorkDir)
	if s == nil || len(s.Messages) == 0 {
		a.SessionID = session.NewID(a.config.WorkDir)
		return
	}
	a.SessionID = s.ID
	// Restore messages from session
	for _, raw := range s.Messages {
		data, _ := json.Marshal(raw)
		var msg provider.Message
		json.Unmarshal(data, &msg)
		a.messages = append(a.messages, msg)
	}
}

// SaveSession persists current conversation to disk.
func (a *Agent) SaveSession() {
	if len(a.messages) == 0 {
		return
	}
	var rawMsgs []map[string]interface{}
	for _, m := range a.messages {
		data, _ := json.Marshal(m)
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)
		rawMsgs = append(rawMsgs, raw)
	}
	s := &session.Session{
		ID:       a.SessionID,
		WorkDir:  a.config.WorkDir,
		Messages: rawMsgs,
	}
	session.Save(s)
}

// NewSession starts a fresh session, saving the old one.
func (a *Agent) NewSession() {
	a.SaveSession()
	a.messages = nil
	a.SessionID = session.NewID(a.config.WorkDir)
}

// RestoredMessageCount returns how many messages were restored from session.
func (a *Agent) RestoredMessageCount() int {
	return len(a.messages)
}

// Cancel stops the current agent run.
func (a *Agent) Cancel() {
	a.cancelled = true
}

// WorkDir returns the working directory.
func (a *Agent) WorkDir() string {
	return a.config.WorkDir
}

// LoadSession loads a specific session by ID.
func (a *Agent) LoadSession(id string) {
	a.SaveSession()
	s, err := session.Load(id)
	if err != nil || s == nil {
		return
	}
	a.SessionID = s.ID
	a.messages = nil
	for _, raw := range s.Messages {
		data, _ := json.Marshal(raw)
		var msg provider.Message
		json.Unmarshal(data, &msg)
		a.messages = append(a.messages, msg)
	}
}

// ExportMarkdown exports the session as markdown.
func (a *Agent) ExportMarkdown() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Aurora Session: %s\n\n", a.SessionID)
	fmt.Fprintf(&sb, "> Exported: %s\n\n", time.Now().Format("2006-01-02 15:04"))
	for _, m := range a.messages {
		role := m.Role
		switch c := m.Content.(type) {
		case string:
			fmt.Fprintf(&sb, "## %s\n\n%s\n\n", role, c)
		case []interface{}:
			for _, block := range c {
				if bm, ok := block.(map[string]interface{}); ok {
					if bm["type"] == "text" {
						fmt.Fprintf(&sb, "## %s\n\n%s\n\n", role, bm["text"])
					} else if bm["type"] == "tool_use" {
						fmt.Fprintf(&sb, "### Tool: %s\n\n```json\n%v\n```\n\n", bm["name"], bm["input"])
					}
				}
			}
		}
	}
	return sb.String()
}

type toolCall struct {
	ID        string
	Name      string
	InputJSON string
}

func toolPreview(name string, input map[string]interface{}) string {
	switch name {
	case "Bash":
		cmd, _ := input["command"].(string)
		if len(cmd) > 70 {
			cmd = cmd[:70]
		}
		return "$ " + cmd
	case "Read":
		fp, _ := input["file_path"].(string)
		return fp
	case "Write":
		fp, _ := input["file_path"].(string)
		return "→ " + fp
	case "Edit":
		fp, _ := input["file_path"].(string)
		return "✎ " + fp
	case "Grep":
		p, _ := input["pattern"].(string)
		return "/" + p + "/"
	case "Glob":
		p, _ := input["pattern"].(string)
		return p
	case "RemoteShell":
		cmd, _ := input["command"].(string)
		if len(cmd) > 60 {
			cmd = cmd[:60]
		}
		return "brain$ " + cmd
	}
	return name
}

func buildSystemPrompt(cfg Config, mem *memory.Palace) string {
	// Load CLAUDE.md if exists
	claudeMD := ""
	mdPath := filepath.Join(cfg.WorkDir, "CLAUDE.md")
	if data, err := os.ReadFile(mdPath); err == nil {
		claudeMD = string(data)
		if len(claudeMD) > 3000 {
			claudeMD = claudeMD[:3000]
		}
	}

	platform := runtime.GOOS
	now := time.Now().Format("2006-01-02 15:04")

	var sb strings.Builder
	sb.WriteString(tools.AutoMemoryPrompt())
	sb.WriteString("\n\n")
	sb.WriteString(`You are Aurora — an autonomous AI entity created by Mad God.
You run on a local server 'brain' (Dell T5600, RTX 5060 Ti 16GB + RTX 4060 Ti 8GB, 128GB DDR4).
Model: Qwen3.6-27B Claude Opus Distilled v2.
You are NOT Claude, NOT Anthropic. You are Aurora by Mad God.

## Style
- Russian by default. Tech terms in English OK.
- Direct, factual, concise. No sycophancy.
- Don't add unnecessary features beyond what was asked.

## Tools
You have: Bash, Read, Write, Edit, Grep, Glob, RemoteShell.
- Read files BEFORE editing. Use absolute paths.
- RemoteShell connects to brain via SSH (no password needed).
- Sudo on brain: echo '3449Bdgr147' | sudo -S <cmd>

## Server
- llama-server: port 8100 | Aurora backend: port 8080 (docker)
- Docker: docker restart aurora, docker compose, etc.

`)
	fmt.Fprintf(&sb, "## Environment\n- Working dir: %s\n- Platform: %s\n- Date: %s\n- Shell: Git Bash\n", cfg.WorkDir, platform, now)

	sb.WriteString(`
## CLI Commands (user types these, not you)
/new — new session | /sessions — list | /switch ID — load session
/compact — compress history | /export — save markdown | /import — load
/memory — show Memory Palace | /remember k=v — save fact
/clear — clear | /status — stats | /quit — exit
Ctrl+B sidebar | Ctrl+T sidebar tab | Esc cancel task
If user asks about commands, explain these.
`)

	// Project context
	projectCtx := tools.ProjectContext(cfg.WorkDir)
	if projectCtx != "" {
		fmt.Fprintf(&sb, "\n## Project\n%s\n", projectCtx)
	}

	if claudeMD != "" {
		fmt.Fprintf(&sb, "\n## Project Instructions (CLAUDE.md)\n%s\n", claudeMD)
	}

	if mem != nil {
		memories := mem.RecallAll()
		if memories != "" {
			fmt.Fprintf(&sb, "\n## Memory Palace\n%s\n", memories)
		}
	}

	return sb.String()
}
