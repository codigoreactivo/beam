# Beam Client — Specification

## Supported Protocols

| Protocol | Port (default) | Library | Notes |
|---|---|---|---|
| SFTP | 22 | `pkg/sftp` + `crypto/ssh` | Preferred, most secure |
| FTP | 21 | `jlaffaye/ftp` | Legacy hosting support |
| FTPS | 990 / 21 | `jlaffaye/ftp` (TLS) | FTP over TLS |

## CLI Interface

### Global flags

```
beam [command] [flags]

Flags:
  --project, -p   string   Target project name
  --config        string   Config path (default: ~/.beam/beam.yaml)
  --quiet, -q              Suppress output except errors
  --json                   Output as JSON (for scripting)
```

### Commands

#### Project management

```bash
beam project list                          # list all projects
beam project add <name>                    # interactive project wizard
beam project add <name> \                  # non-interactive
  --host ftp.mysite.com \
  --user admin \
  --proto sftp \
  --local ./dist \
  --remote /public_html

beam project remove <name>                 # remove a project
beam project edit <name>                   # open project in editor
beam project show <name>                   # show project details
beam project test <name>                   # test connection
```

#### File operations

```bash
beam upload <local> [remote] -p <project>  # upload file or directory
beam download <remote> [local] -p <project># download file or directory
beam ls [remote-path] -p <project>         # list remote directory
beam rm <remote-path> -p <project>         # delete remote file/dir
beam mv <src> <dst> -p <project>           # rename/move remote file
beam mkdir <remote-path> -p <project>      # create remote directory
beam diff -p <project>                     # diff local vs remote
```

#### Sync and deploy

```bash
beam sync -p <project>                     # sync local → remote (incremental)
beam sync --dry-run -p <project>           # preview what would sync
beam deploy -p <project>                   # full deploy (sync + hooks)
beam watch -p <project>                    # watch local dir and auto-sync
beam watch --once -p <project>             # sync once then exit
```

#### Logs

```bash
beam logs -p <project>                     # tail project logs
beam logs --all                            # tail all project logs
beam logs --export -p <project>            # export logs to file
```

#### Interfaces

```bash
beam tui                                   # launch TUI dashboard
beam gui                                   # launch desktop GUI
```

## TUI Interface

Built with Bubbletea + Lip Gloss.

### Layout

```
┌─ Beam ──────────────────────────────────────────────────────────┐
│ [Tab] Switch project  [U] Upload  [D] Download  [W] Watch  [Q] Quit│
├──────────────────┬──────────────────────────────────────────────┤
│ PROJECTS         │ mysite-prod                         ● Connected│
│                  ├──────────────────────────────────────────────┤
│ ● mysite-prod    │ LOCAL              REMOTE                     │
│ ○ mysite-staging │ /dist              /var/www/html              │
│ ○ client-a       ├──────────────────────────────────────────────┤
│ ○ client-b       │ 📁 assets/         📁 assets/                 │
│                  │ 📁 js/             📁 js/                     │
│ [N] New project  │ 📄 index.html  →   📄 index.html  ✓          │
│ [E] Edit         │ 📄 about.html  →   📄 about.html  ✓          │
│ [X] Remove       │ 📄 contact.html    — (not uploaded)           │
│                  ├──────────────────────────────────────────────┤
│                  │ ACTIVITY LOG                                  │
│                  │ 14:32:01  ✓ Uploaded index.html (2.3KB)      │
│                  │ 14:32:00  ✓ Uploaded main.js (45KB)          │
│                  │ 14:31:58  ⚡ Hook: cache cleared              │
├──────────────────┴──────────────────────────────────────────────┤
│ 2 projects active  │  Last upload: 14:32:01  │  Watch: ON       │
└─────────────────────────────────────────────────────────────────┘
```

### Keyboard shortcuts

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Switch between projects |
| `U` | Upload selected / sync |
| `D` | Download selected |
| `W` | Toggle file watcher |
| `N` | New project wizard |
| `E` | Edit current project |
| `L` | View full logs |
| `?` | Help |
| `Q` | Quit |

## GUI Interface (Wails + React + Tailwind v4)

### Design principles
- Minimalist, dark-mode first
- Sidebar for project list
- Split panel: local tree (left) / remote tree (right)
- Drag and drop upload
- Activity feed at the bottom
- Status bar always visible

### Pages

| Page | Description |
|---|---|
| Dashboard | Overview of all projects, recent activity |
| Project | File manager split view for one project |
| Settings | Global config, default protocol, MCP port |
| Logs | Structured log viewer with filters |

## File Watcher Behavior

- Debounce: 500ms after last change before triggering upload
- Ignore patterns: `.git/`, `node_modules/`, `*.tmp`, `*.log`
- Configurable per project via `.beamignore` file (same syntax as `.gitignore`)
- Recursive by default, configurable depth

## Hook Configuration

```yaml
hooks:
  pre-upload:
    - type: shell
      cmd: npm run build
      blocking: true        # upload waits for this to complete

  post-upload:
    - type: shell
      cmd: echo "Done"
    - type: webhook
      url: https://example.com/hook
      method: POST
      body: '{"project": "mysite"}'
    - type: notification    # desktop notification
      title: "Beam"
      message: "mysite deployed"

  on-error:
    - type: shell
      cmd: ./notify-error.sh
```

## Credentials Security

- Passwords stored in OS keychain (via `zalando/go-keyring`)
- SSH keys referenced by path, never copied
- Environment variable interpolation in config (`${VAR_NAME}`)
- Optional: `.env` file per project directory

## Transfer Queue

- Configurable concurrent workers per project (default: 3)
- Failed transfers auto-retry (default: 3 attempts, exponential backoff)
- Queue persisted to disk for resumability on crash
- Priority queue: smaller files first (configurable)
