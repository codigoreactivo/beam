# Beam MCP Tools

Beam exposes an MCP server that allows Claude (or any MCP-compatible LLM client) to fully operate the FTP client. The MCP server runs as a local process alongside Beam.

## Starting the MCP Server

```bash
beam mcp serve                    # starts on default port 7071
beam mcp serve --port 8080        # custom port
```

Configure in Claude Desktop or any MCP client:
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

---

## Tools Reference

### Project Management

#### `beam_project_list`
Returns all configured projects with their status.

```json
Input:  {}
Output: {
  "projects": [
    {
      "name": "mysite-prod",
      "protocol": "sftp",
      "host": "mysite.com",
      "env": "production",
      "connected": true,
      "watching": false,
      "last_deploy": "2024-01-15T14:32:00Z"
    }
  ]
}
```

#### `beam_project_add`
Adds a new project profile.

```json
Input: {
  "name": "mysite-prod",
  "protocol": "sftp",         // sftp | ftp | ftps
  "host": "mysite.com",
  "port": 22,
  "user": "deploy",
  "password": "...",          // optional if using key
  "key_path": "~/.ssh/id_rsa",// optional
  "local": "./dist",
  "remote": "/var/www/html",
  "env": "production"         // dev | staging | production
}
Output: { "success": true, "project": { ... } }
```

#### `beam_project_remove`
Removes a project profile.

```json
Input:  { "name": "mysite-prod" }
Output: { "success": true }
```

#### `beam_project_test`
Tests connectivity for a project.

```json
Input:  { "name": "mysite-prod" }
Output: { "reachable": true, "latency_ms": 42, "server_info": "OpenSSH_8.9" }
```

---

### File Operations

#### `beam_upload`
Uploads a local file or directory to a remote path.

```json
Input: {
  "project": "mysite-prod",
  "local": "./dist/index.html",
  "remote": "/var/www/html/index.html",
  "overwrite": true
}
Output: {
  "success": true,
  "bytes_transferred": 4096,
  "duration_ms": 120
}
```

#### `beam_download`
Downloads a remote file to a local path.

```json
Input: {
  "project": "mysite-prod",
  "remote": "/var/www/html/config.php",
  "local": "./backup/config.php"
}
Output: { "success": true, "bytes_transferred": 1024 }
```

#### `beam_list`
Lists contents of a remote directory.

```json
Input: {
  "project": "mysite-prod",
  "path": "/var/www/html",
  "recursive": false
}
Output: {
  "entries": [
    { "name": "index.html", "size": 4096, "modified": "2024-01-15T10:00:00Z", "type": "file" },
    { "name": "assets", "type": "directory" }
  ]
}
```

#### `beam_delete`
Deletes a remote file or directory.

```json
Input: {
  "project": "mysite-prod",
  "path": "/var/www/html/old-page.html",
  "recursive": false
}
Output: { "success": true }
```

#### `beam_move`
Renames or moves a remote file.

```json
Input: {
  "project": "mysite-prod",
  "from": "/var/www/html/page-old.html",
  "to": "/var/www/html/page.html"
}
Output: { "success": true }
```

---

### Sync and Deploy

#### `beam_diff`
Returns the diff between local and remote state.

```json
Input: { "project": "mysite-prod" }
Output: {
  "to_upload": ["contact.html", "assets/new-logo.png"],
  "to_update": ["index.html"],
  "remote_only": ["old-page.html"],
  "in_sync": 42
}
```

#### `beam_sync`
Syncs local directory to remote (incremental).

```json
Input: {
  "project": "mysite-prod",
  "dry_run": false,
  "delete_remote": false    // remove remote files not present locally
}
Output: {
  "uploaded": 3,
  "updated": 1,
  "skipped": 42,
  "errors": []
}
```

#### `beam_deploy`
Full deploy: runs pre-hooks, syncs, runs post-hooks.

```json
Input: {
  "project": "mysite-prod",
  "dry_run": false
}
Output: {
  "success": true,
  "hooks_run": ["pre-deploy: npm run build", "post-deploy: clear-cache"],
  "files_transferred": 4,
  "duration_ms": 3200
}
```

---

### File Watcher

#### `beam_watch_start`
Starts the file watcher for a project.

```json
Input: { "project": "mysite-prod" }
Output: { "watching": true }
```

#### `beam_watch_stop`
Stops the file watcher for a project.

```json
Input: { "project": "mysite-prod" }
Output: { "watching": false }
```

#### `beam_watch_status`
Returns watcher status across all projects.

```json
Input: {}
Output: {
  "watchers": [
    { "project": "mysite-prod", "watching": true, "events_today": 47 },
    { "project": "mysite-staging", "watching": false }
  ]
}
```

---

### Logs and Monitoring

#### `beam_logs`
Returns recent log entries for a project.

```json
Input: {
  "project": "mysite-prod",
  "limit": 20,
  "level": "info",          // debug | info | warn | error
  "since": "2024-01-15T00:00:00Z"
}
Output: {
  "entries": [
    {
      "time": "2024-01-15T14:32:01Z",
      "level": "info",
      "message": "Uploaded index.html",
      "meta": { "size": 4096, "duration_ms": 120 }
    }
  ]
}
```

#### `beam_stats`
Returns transfer statistics for a project.

```json
Input: {
  "project": "mysite-prod",
  "period": "7d"            // 1d | 7d | 30d
}
Output: {
  "total_uploads": 142,
  "total_bytes": 15728640,
  "avg_duration_ms": 340,
  "error_rate": 0.02,
  "most_uploaded": "index.html"
}
```

---

### Hook Management

#### `beam_hook_run`
Manually triggers a hook for a project.

```json
Input: {
  "project": "mysite-prod",
  "event": "post-deploy"
}
Output: {
  "hooks_run": 2,
  "results": [
    { "type": "shell", "cmd": "clear-cache.sh", "exit_code": 0 },
    { "type": "webhook", "url": "...", "status": 200 }
  ]
}
```

---

## Resources

MCP resources expose read-only state for Claude to inspect:

| Resource | Description |
|---|---|
| `beam://projects` | All project profiles |
| `beam://projects/{name}` | Single project details |
| `beam://projects/{name}/logs` | Project logs |
| `beam://projects/{name}/stats` | Project statistics |
| `beam://config` | Global Beam configuration |

---

## Future: Server MCP Tools (beam-server)

When targeting a Beam Server project, additional tools become available:

| Tool | Description |
|---|---|
| `beam_server_status` | Server health and uptime |
| `beam_server_connections` | Active client connections |
| `beam_server_user_list` | All server users |
| `beam_server_user_add` | Add a server user |
| `beam_server_user_remove` | Remove a server user |
| `beam_server_kick` | Disconnect a user session |
| `beam_server_restart` | Restart the server process |
