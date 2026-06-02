# Beam — Architecture

## Overview

Beam is structured as a single Go binary that exposes four interfaces over a shared core engine. The architecture is designed so the client works standalone today, and a managed server component can be added in the future without breaking existing integrations.

```
┌─────────────────────────────────────────────────────────────┐
│                        INTERFACES                           │
│                                                             │
│   CLI (Cobra)    TUI (Bubbletea)    GUI (Wails+React)       │
│        │               │                  │                 │
│        └───────────────┴──────────────────┘                 │
│                        │                                    │
│               MCP Server (mark3labs/mcp-go)                 │
│                        │                                    │
└────────────────────────┼────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                      CORE ENGINE                            │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │Project Manager│  │Transfer Engine│  │  Hook Engine     │  │
│  │              │  │              │  │                  │  │
│  │ - profiles   │  │ - FTP client │  │ - pre-upload     │  │
│  │ - credentials│  │ - SFTP client│  │ - post-upload    │  │
│  │ - N projects │  │ - FTPS client│  │ - on-connect     │  │
│  │ - state      │  │ - queue      │  │ - on-disconnect  │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  File Watcher│  │  Log Engine  │  │  Config Manager  │  │
│  │              │  │              │  │                  │  │
│  │ - watchdog   │  │ - per project│  │ - ~/.beam/       │  │
│  │ - debounce   │  │ - structured │  │ - projects.yaml  │  │
│  │ - triggers   │  │ - exportable │  │ - beam.yaml      │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   REMOTE SERVERS                            │
│                                                             │
│    FTP Server         SFTP Server         FTPS Server       │
│   (port 21)           (port 22)           (port 990)        │
│   your hosting        your VPS            legacy servers    │
│                                                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              FUTURE: BEAM SERVER (separate binary)          │
│                                                             │
│  beam-server — self-hosted FTP/SFTP server                  │
│  Managed via Beam client using the same project system      │
│  Exposes same MCP tools for full automation                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Component Descriptions

### Project Manager
Central registry of all connection profiles. Each project holds:
- Protocol (FTP / SFTP / FTPS)
- Host, port, credentials
- Local root directory
- Remote root directory
- Environment (dev, staging, prod)
- Hook definitions

### Transfer Engine
Handles all file operations against remote servers:
- Upload single file or directory tree
- Download files from remote
- Delete, rename, move on remote
- Directory listing and diffing
- Resumable transfers (where protocol allows)
- Concurrent transfer queue with configurable workers

### Hook Engine
Event-driven automation layer:
- Triggers: `pre-upload`, `post-upload`, `pre-deploy`, `post-deploy`, `on-connect`, `on-error`
- Actions: shell commands, HTTP webhooks, notifications
- Per-project hook configuration

### File Watcher
Monitors local directories for changes:
- Debounced events (avoids upload storms on saves)
- Glob-based include/exclude rules
- Triggers Transfer Engine automatically

### MCP Server
Exposes Beam's capabilities to Claude and other LLM clients:
- All project operations as MCP tools
- Read-only project inspection as MCP resources
- Enables conversational workflow automation

## Data Flow — Upload Example

```
User saves file locally
        │
        ▼
File Watcher detects change
        │
        ▼
Hook Engine: fires pre-upload hooks
        │
        ▼
Transfer Engine: queues upload job
        │
        ▼
Protocol client (SFTP/FTP/FTPS) executes transfer
        │
        ▼
Hook Engine: fires post-upload hooks (e.g. clear cache, notify Slack)
        │
        ▼
Log Engine: records transfer result
        │
        ▼
TUI/GUI/CLI: updates display
```

## Configuration Structure

```
~/.beam/
├── beam.yaml          # global config (default protocol, log level, MCP port)
├── projects.yaml      # all project profiles
├── logs/
│   ├── mysite.log
│   └── staging.log
└── keys/              # SSH keys per project (optional)
    └── mysite_rsa
```

### projects.yaml example

```yaml
projects:
  - name: mysite-prod
    protocol: sftp
    host: mysite.com
    port: 22
    user: deploy
    key: ~/.beam/keys/mysite_rsa
    local: /Users/me/projects/mysite/dist
    remote: /var/www/html
    env: production
    hooks:
      post-upload:
        - type: shell
          cmd: echo "Deployed to prod"
        - type: webhook
          url: https://hooks.mysite.com/deploy

  - name: mysite-staging
    protocol: ftp
    host: staging.mysite.com
    port: 21
    user: admin
    password: "${STAGING_FTP_PASS}"
    local: /Users/me/projects/mysite/dist
    remote: /public_html
    env: staging
```

## Multi-Project Concurrency Model

```
Main process
├── goroutine: MCP server (always on, port configurable)
├── goroutine: TUI render loop (if tui mode)
├── goroutine: GUI bridge (if gui mode)
└── per-project goroutines (spawned on connect):
    ├── project "mysite-prod"
    │   ├── goroutine: file watcher
    │   ├── goroutine: transfer queue worker(s)
    │   └── goroutine: connection keepalive
    └── project "mysite-staging"
        ├── goroutine: file watcher
        ├── goroutine: transfer queue worker(s)
        └── goroutine: connection keepalive
```

## Future: Beam Server

The server is designed as a separate binary (`beam-server`) that:
- Implements an FTP/SFTP server using the same project/config conventions
- Can be managed by the Beam client as a special project type
- Exposes the same MCP tools so Claude can manage server-side operations
- Shares the `projects.yaml` schema so client and server configs are compatible

The client is built today with the server contracts in mind:
- Protocol abstraction layer allows adding server-side protocol handlers
- Hook engine is symmetric (client and server will share the same hook format)
- MCP tools are designed to work whether the remote is an external host or a Beam server
