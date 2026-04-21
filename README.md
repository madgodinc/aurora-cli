# Aurora CLI

Terminal client for [Aurora AI](https://fraylon.net) — free AI coding assistant by Mad God Inc.

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

## Features

- Interactive chat with streaming
- Local file/folder access (mention paths in chat)
- Project management (create, files, download)
- Cross-platform sessions (CLI + Telegram sync)
- File approval system (ask/auto modes)
- Animated thinking spinner
- Browser-based Telegram auth

## Requirements

- Python 3.8+
- `httpx` (installed automatically)

## License

MIT - Mad God Inc. 2026