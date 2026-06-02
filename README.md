# Beam

> Beam up your files. A modern SFTP/FTP client with CLI, TUI, GUI, and MCP support.

Beam is a developer-friendly FTP/SFTP client built in Go. It replaces tools like FileZilla with a faster, scriptable, and AI-assisted experience — four interfaces, one binary, zero runtime dependencies.

---

## Interfaces

| Interface | Command | Use case |
|---|---|---|
| **CLI** | `beam deploy -p prod` | Scripting, CI/CD, automation |
| **TUI** | `beam tui` | Interactive terminal dashboard *(coming soon)* |
| **GUI** | `beam gui` | Desktop app, non-technical users *(coming soon)* |
| **MCP** | `beam mcp serve` | Claude AI-assisted workflows |

---

## Quick Start

```bash
# 1. Add a project
beam project add mysite \
  --host mysite.com --user deploy --proto sftp \
  --key ~/.ssh/id_rsa --local ./dist --remote /var/www/html \
  --env production

# 2. Test the connection
beam project test mysite

# 3. See what's out of sync
beam diff -p mysite

# 4. Deploy (pre-hooks + sync + post-hooks)
beam deploy -p mysite

# 5. Watch for changes and auto-upload
beam watch -p mysite
```

---

## CLI Reference

### Project management

```bash
beam project list                        # list all projects
beam project add <name> [flags]          # add project (interactive wizard if no flags)
beam project show <name>                 # show project details
beam project remove <name>               # remove project
beam project test <name>                 # test live connection
beam project edit <name>                 # open projects.yaml in $EDITOR
```

**`project add` flags:**
```
--host      Remote host (required)
--user      Remote user (required)
--proto     Protocol: sftp (default) | ftp | ftps
--port      Port (default: protocol default — 22/21/990)
--key       Path to SSH private key (SFTP)
--local     Local directory (default: ./)
--remote    Remote directory (default: /)
--env       Environment label: dev | staging | production
```

### File operations

```bash
beam ls [path] -p <project>              # list remote directory
beam upload <local> [remote] -p <project>  # upload file or directory
beam download <remote> [local] -p <project># download file
beam rm <remote-path> -p <project>       # delete remote file/dir
beam mv <src> <dst> -p <project>         # rename/move remote file
beam mkdir <remote-path> -p <project>    # create remote directory
```

### Sync and deploy

```bash
beam diff -p <project>                   # diff local vs remote (no upload)
beam sync -p <project>                   # incremental sync local → remote
beam sync --dry-run -p <project>         # preview sync without transferring
beam deploy -p <project>                 # pre-hooks + sync + post-hooks
beam watch -p <project>                  # watch and auto-upload on file change
beam watch --once -p <project>           # sync once then exit
```

### Logs

```bash
beam logs -p <project>                   # show last 50 lines
beam logs -p <project> -n 100            # show last 100 lines
beam logs -p <project> -f                # follow (tail -f)
beam logs --all                          # all projects
beam logs -p <project> --export          # export to file
```

### MCP server

```bash
beam mcp serve                           # start MCP server (stdio transport)
```

### Global flags

```
-p, --project   Target project name
    --config    Config file path (default: ~/.beam/beam.yaml)
-q, --quiet     Suppress output except errors
    --json      Output as JSON (for scripting)
```

---

## Configuration

Beam stores all config in `~/.beam/`:

```
~/.beam/
├── beam.yaml          # global settings
├── projects.yaml      # all project profiles
├── logs/
│   └── mysite.log
└── keys/              # SSH keys (optional)
```

### `~/.beam/projects.yaml`

```yaml
projects:
  - name: mysite-prod
    protocol: sftp
    host: mysite.com
    port: 22
    user: deploy
    key: ~/.ssh/id_rsa
    local: /Users/me/projects/mysite/dist
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

### `~/.beam/beam.yaml`

```yaml
default_protocol: sftp
log_level: info
mcp_port: 7071
workers: 3
```

---

## Hooks

Hooks fire at key events in the deploy/upload lifecycle:

| Event | When |
|---|---|
| `pre-deploy` | Before sync starts |
| `post-deploy` | After sync completes |
| `pre-upload` | Before each file upload |
| `post-upload` | After each file upload |
| `on-connect` | On server connection |
| `on-error` | On any error |

**Hook types:**

```yaml
# Shell command
- type: shell
  cmd: npm run build
  blocking: true     # if true, failure stops the deploy

# HTTP webhook
- type: webhook
  url: https://example.com/hook
  method: POST
  body: '{"event": "deploy", "project": "mysite"}'

# Desktop notification (macOS)
- type: notification
  title: "Beam"
  message: "Deployed successfully"
```

---

## MCP Integration (Claude Desktop)

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

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

**Available MCP tools:** `beam_project_list`, `beam_project_add`, `beam_project_remove`, `beam_project_test`, `beam_list`, `beam_upload`, `beam_download`, `beam_delete`, `beam_move`, `beam_diff`, `beam_sync`, `beam_deploy`, `beam_hook_run`, `beam_logs`

**MCP resources:** `beam://projects`, `beam://config`

---

## Sync behavior

`beam sync` compares local vs remote using **modification time + file size** (same approach as rsync default mode):

- Local file not on remote → **upload**
- Local newer than remote (1s tolerance) or different size → **upload**
- Same size and mtime → **skip**

**Default ignores:** `.git/`, `node_modules/`, `.svn/`, `.hg/`, `*.tmp`, `*.log`, `*.swp`

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| CLI | [Cobra](https://github.com/spf13/cobra) |
| TUI | [Bubbletea](https://github.com/charmbracelet/bubbletea) + Lip Gloss *(coming)* |
| GUI | [Wails v2](https://wails.io) + React + Tailwind v4 *(coming)* |
| MCP | [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) |
| SFTP | [pkg/sftp](https://github.com/pkg/sftp) + crypto/ssh |
| FTP/FTPS | [jlaffaye/ftp](https://github.com/jlaffaye/ftp) *(coming)* |
| File watch | [fsnotify](https://github.com/fsnotify/fsnotify) |
| Config | gopkg.in/yaml.v3 |

---

## Roadmap

- [x] CLI core + all commands wired (Cobra)
- [x] Project manager (`~/.beam/projects.yaml`)
- [x] SFTP client — upload, download, ls, rm, mv, mkdir
- [x] Sync engine — incremental diff + upload
- [x] File watcher — fsnotify + debounce 500ms
- [x] Hook engine — shell, webhook, notification
- [x] `beam deploy` — hooks + sync pipeline
- [x] `beam logs` — tail, follow, export
- [x] MCP server — 14 tools + 2 resources (Claude Desktop ready)
- [ ] FTP/FTPS client (jlaffaye/ftp)
- [ ] OS keychain for credentials (zalando/go-keyring)
- [ ] `.beamignore` per-project ignore patterns
- [ ] TUI dashboard (Bubbletea + Lip Gloss)
- [ ] GUI desktop app (Wails + React + Tailwind v4)
- [ ] Concurrent transfer queue with retry/backoff
- [ ] SSH known_hosts verification
- [ ] `beam-server` (future — self-hosted SFTP server)

---

## Documentation

- [Architecture](docs/architecture.md) — Full system design
- [Client Spec](docs/client-spec.md) — CLI, TUI, GUI spec with layouts
- [Server Spec](docs/server-spec.md) — Future server spec
- [Use Cases](docs/use-cases.md) — 15 use cases
- [MCP Tools](docs/mcp-tools.md) — All MCP tools with schemas
