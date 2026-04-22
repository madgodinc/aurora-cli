# Aurora CLI

Terminal client for [Aurora AI](https://fraylon.net) — free AI coding assistant by Mad God Inc.

1050+ skills. 35 tools. Code analysis, SQL queries on CSV, security scanning, QR codes, web search, vision, and more. Self-hosted, private, free.

## Install

```bash
pip install git+https://github.com/madgodinc/aurora-cli.git
```

## Usage

```bash
aurora                          # interactive chat
aurora "write hello world"      # single request
```

## Commands

### Chat & Projects
| Command | Description |
|---------|-------------|
| `/help` | Show help |
| `/project` | List/connect server projects |
| `/open PATH` | Open local folder |
| `/close` | Close local project |
| `/files` | List project files |
| `/session new` | Create cross-platform session |
| `/session ID` | Connect to session (sync with TG) |
| `/approve auto` | Auto-apply file changes |
| `/approve ask` | Ask before each change |
| `/health` | Server health check |
| `/login` | Auth via browser |
| `/quit` | Exit |

### Local Storage
| Command | Description |
|---------|-------------|
| `/memory` | Show local memory (stored facts) |
| `/memory key=val` | Save a fact locally |
| `/compress` | Compress session context (summary) |
| `/export` | Export session to Markdown |
| `/history` | Show last 10 messages |
| `/sessions` | List local sessions |

## Features

- Interactive chat with animated spinner
- **Local vault** (`~/.aurora/`) — all history and memory saved locally on your PC
- **Auto-save** — every message automatically saved to local session
- Local file/folder access (mention paths in chat)
- Project management (create, files, download)
- Cross-platform sessions (CLI + Telegram sync)
- File approval system (ask/auto modes)
- Vision: auto-detect image paths in messages
- Browser-based Telegram auth
- Obsidian-compatible format (`.md` + `.jsonl`)

## Local Storage

All data stored in `~/.aurora/`:

```
~/.aurora/
├── config.json       # API key, settings
├── context.json      # Compressed session context
├── sessions/         # Chat history (JSONL)
│   ├── default.jsonl
│   └── <session_id>.jsonl
└── memory/
    └── facts.json    # Your saved facts
```

Open `~/.aurora/` in Obsidian to browse as a knowledge base.

## Requirements

- Python 3.8+
- `httpx` (installed automatically)

## License

MIT - Mad God Inc. 2026