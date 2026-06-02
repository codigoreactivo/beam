# Beam

> Beam up your files. A modern SFTP/FTP client — CLI, TUI, GUI, and MCP.

Beam is a developer-friendly file transfer tool built in Go. It replaces tools like FileZilla with a faster, scriptable, and AI-assisted experience: four interfaces, one binary, zero runtime dependencies.

## Features

- **Multi-protocol** — SFTP, FTP, and FTPS in a single tool
- **Project profiles** — Named connection profiles with env labels (dev/staging/prod)
- **Incremental sync** — rsync-style mtime + size comparison with 1s tolerance
- **Deploy pipeline** — pre-hooks → sync → post-hooks in one command
- **File watching** — fsnotify-based watcher with 500ms debounce and auto-upload
- **Hook engine** — Shell commands, webhooks, and desktop notifications at 6 lifecycle events
- **Config migration** — Import/export from FileZilla, Cyberduck, CoreFTP, and .env
- **OS keychain** — Passwords stored in the system keychain with YAML fallback
- **SSH known_hosts** — TOFU (Trust On First Use) with MITM detection
- **Retry with backoff** — Exponential backoff on transient upload failures
- **Ignore patterns** — `.beamignore` file (gitignore-style) + built-in defaults
- **JSON output** — `--json` flag for scripting and automation

## Quick Start

```bash
# Add a project
beam project add mysite \
  --host mysite.com --user deploy --proto sftp \
  --key ~/.ssh/id_rsa --local ./dist --remote /var/www/html

# Test the connection
beam project test mysite

# See what's out of sync
beam diff -p mysite

# Deploy (pre-hooks + sync + post-hooks)
beam deploy -p mysite

# Watch for changes and auto-upload
beam watch -p mysite
```

## Interfaces

| Interface | Command | Use case |
|-----------|---------|----------|
| **CLI** | `beam deploy -p prod` | Scripting, CI/CD, automation |
| **TUI** | `beam tui` | Interactive terminal dashboard |
| **GUI** | `beam gui` | Desktop app (Wails + React) |
| **MCP** | `beam mcp serve` | Claude AI-assisted workflows |

## CLI Reference

```
beam
├── project           Manage connection profiles
│   ├── list          List all projects
│   ├── add <name>    Add a project (interactive or --flags)
│   ├── remove <name> Remove a project
│   ├── show <name>   Show project details
│   ├── edit <name>   Open projects.yaml in $EDITOR
│   ├── test <name>   Test live connection
│   ├── import <file> Import from FileZilla/Cyberduck/CoreFTP/.env
│   └── export [name] Export to beam/filezilla/env format
│
├── ls [path]         List remote directory
├── upload <l> [r]    Upload file or directory
├── download <r> [l]  Download file
├── rm <path>         Delete remote file/directory
├── mv <src> <dst>    Rename/move remote file
├── mkdir <path>      Create remote directory
├── diff              Show local vs remote diff
├── sync              Incremental sync local → remote (--dry-run)
├── deploy            Pre-hooks + sync + post-hooks pipeline
├── watch             Watch and auto-upload (--once)
├── logs              View and follow project logs (--follow, --export)
├── mcp serve         Start MCP server (stdio transport)
├── tui               Launch TUI dashboard
└── gui               Launch GUI desktop app
```

## Configuration

Beam stores all config in `~/.beam/`:

```
~/.beam/
├── beam.yaml          # Global settings
├── projects.yaml      # All project profiles
├── logs/
│   └── mysite.log
└── keys/              # SSH keys
```

### Global config (`~/.beam/beam.yaml`)

```yaml
default_protocol: sftp
log_level: info
mcp_port: 7071
workers: 3
```

### Project profile (`~/.beam/projects.yaml`)

```yaml
projects:
  - name: mysite-prod
    protocol: sftp
    host: mysite.com
    port: 22
    user: deploy
    key: ~/.ssh/id_rsa
    local: ./dist
    remote: /var/www/html
    env: production
    hooks:
      pre-deploy:
        - type: shell
          cmd: npm run build
          blocking: true
      post-deploy:
        - type: webhook
          url: https://hooks.mysite.com/deploy
        - type: notification
          title: "Beam"
          message: "mysite deployed"
```

## Hooks

Hooks fire at key events in the deploy/upload lifecycle:

| Event | When |
|-------|------|
| `pre-deploy` | Before sync starts |
| `post-deploy` | After sync completes |
| `pre-upload` | Before each file upload |
| `post-upload` | After each file upload |
| `on-connect` | On server connection |
| `on-error` | On any error |

**Hook types:** shell commands, HTTP webhooks, desktop notifications (macOS).

## MCP Integration

Add to your Claude Desktop config:

```json
{
  "mcpServers": {
    "beam": {
      "command": "beam",
      "args": ["mcp", "serve"]
    }
  }
}
```

Claude can then manage your deployments conversationally:

> *"Deploy mysite-prod"*
> *"Show me what files changed since last sync"*
> *"Upload just the CSS files to staging"*

**14 MCP tools** · **2 MCP resources** (`beam://projects`, `beam://config`)

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 |
| CLI | [Cobra](https://github.com/spf13/cobra) |
| TUI | [Bubbletea](https://github.com/charmbracelet/bubbletea) + Lip Gloss |
| GUI | [Wails v2](https://wails.io) + React + Tailwind v4 |
| MCP | [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) |
| SFTP | [pkg/sftp](https://github.com/pkg/sftp) + crypto/ssh |
| FTP/FTPS | [jlaffaye/ftp](https://github.com/jlaffaye/ftp) |
| File watch | [fsnotify](https://github.com/fsnotify/fsnotify) |
| Keychain | [zalando/go-keyring](https://github.com/zalando/go-keyring) |

## Documentation

- [Architecture](docs/architecture.md)
- [Client Spec](docs/client-spec.md)
- [Server Spec](docs/server-spec.md)
- [Use Cases](docs/use-cases.md)
- [MCP Tools](docs/mcp-tools.md)
