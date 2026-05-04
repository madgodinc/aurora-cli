# Aurora CLI

**Autonomous AI Agent for your terminal.** Built by [Mad God Inc.](https://github.com/madgodinc)

Aurora CLI is a full-featured AI coding assistant that runs in your terminal. It connects to any Anthropic-compatible API (local models via llama.cpp, or cloud providers) and gives you an AI agent with real tools — file editing, shell commands, web search, git, and more.

## Quick Install

**Windows (Git Bash):**
```bash
curl -sL https://raw.githubusercontent.com/madgodinc/aurora-cli/main/install.sh | bash
```

**After install, just type:**
```bash
aurora
```

## Features

- **16 tools**: Bash, Read, Write, Edit, Grep, Glob, Git, RemoteShell, Docker, ServerStatus, ModelSwitch, WebSearch, WebFetch, Clipboard, Process, Remember
- **Memory Palace** — persistent memory across sessions, per-user isolation
- **Auto-memory** — Aurora automatically remembers important facts
- **Streaming** — real-time response streaming with markdown rendering
- **TUI** — beautiful terminal UI with Bubble Tea (sidebar, tool activity panel, chat)
- **Project detection** — auto-scans go.mod, package.json, .git, CLAUDE.md
- **TG Auth** — authenticate via Telegram bot
- **Multi-server** — connect to any Anthropic-compatible API
- **Single binary** — one .exe, no dependencies

## Setup

On first run, Aurora will ask you to configure:

1. **Connection** — local server, internet (llm.fraylon.net), or your own server
2. **Authentication** — auto-detected via SSH (owner), or via Telegram
3. **Memory** — automatically initialized per-user

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+B` | Toggle sidebar |
| `Ctrl+C` | Quit |
| `Enter` | Send message |

## Commands

| Command | Description |
|---------|-------------|
| `/help` | Show help |
| `/compact` | Compress conversation history |
| `/memory` | Show Memory Palace |
| `/remember key = value` | Save to memory |
| `/clear` | Clear chat |
| `/status` | Show stats |
| `/quit` | Exit |

## Architecture

```
aurora.exe (17MB single binary)
├── TUI (Bubble Tea + Lip Gloss + Glamour)
├── Agent (tool loop: stream → tool_use → execute → tool_result)
├── Provider (Anthropic Messages API streaming client)
├── Tools (16 built-in tools)
├── Memory Palace (persistent per-user JSON)
├── Config (setup wizard, TG auth, multi-server)
└── Session (save/resume conversations)
```

## License

MIT — use however you want.

## Credits

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling
- [Glamour](https://github.com/charmbracelet/glamour) — markdown rendering

(c) 2026 Mad God Inc.
