# Beam

> Beam up your files. A modern FTP/SFTP client with TUI, CLI, GUI, and MCP support.

Beam is a developer-friendly and user-friendly FTP/SFTP client built in Go. It replaces tools like FileZilla with a faster, scriptable, and AI-assisted experience across multiple interfaces.

## Interfaces

| Interface | Command | Use case |
|---|---|---|
| CLI | `beam deploy prod` | Scripting, CI/CD, automation |
| TUI | `beam tui` | Interactive terminal dashboard |
| GUI | `beam gui` | Visual management, non-technical users |
| MCP | — | Claude AI-assisted workflows |

## Key Features

- **Multi-project**: Manage unlimited FTP/SFTP connections simultaneously
- **Protocol support**: FTP, FTPS, SFTP
- **Automation**: File watchers, deploy hooks, triggers
- **MCP integration**: Let Claude operate Beam on your behalf
- **Single binary**: No runtime dependencies, works everywhere

## Quick Start

```bash
# Add a project
beam project add mysite --host ftp.mysite.com --user admin --proto sftp

# Upload a file
beam upload ./dist prod

# Watch and sync on change
beam watch mysite --local ./src --remote /public_html

# Open interactive TUI
beam tui

# Open desktop GUI
beam gui
```

## Documentation

- [Architecture](docs/architecture.md) — Full system design, client + future server
- [Client Spec](docs/client-spec.md) — Detailed client specification
- [Server Spec](docs/server-spec.md) — Future server implementation
- [Use Cases](docs/use-cases.md) — All use cases by interface
- [MCP Tools](docs/mcp-tools.md) — AI-assisted workflow tools

## Roadmap

- [x] Documentation and architecture design
- [ ] CLI core + project management
- [ ] SFTP/FTP client engine
- [ ] TUI dashboard (Bubbletea + Lip Gloss)
- [ ] MCP server integration
- [ ] GUI desktop app (Wails + React + Tailwind v4)
- [ ] File watcher + auto-deploy
- [ ] Server implementation (future)

## Stack

- **Language**: Go
- **TUI**: Bubbletea + Lip Gloss (Charm.sh)
- **CLI**: Cobra
- **GUI**: Wails + React + Tailwind v4
- **MCP**: mark3labs/mcp-go
- **Protocols**: FTP (jlaffaye/ftp), SFTP (pkg/sftp + crypto/ssh)
