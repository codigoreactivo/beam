# Beam Server — Specification (Future Implementation)

> Status: Documented, not yet implemented.
> The client is built with server compatibility in mind.

## Overview

`beam-server` is a self-hosted FTP/SFTP server that can be managed by the Beam client as a first-class project type. It shares the same config schema, hook format, and MCP tools as the client.

## Why a Beam Server

| Scenario | Without Beam Server | With Beam Server |
|---|---|---|
| Self-hosted VPS | Use existing vsftpd/OpenSSH | Managed FTP/SFTP with Beam conventions |
| Team collaboration | Individual FTP accounts | Centralized project-aware server |
| Automation | Manual hook setup on server | Hooks configured from client |
| AI management | Client-side only | Claude can manage server + client end-to-end |

## Architecture

```
beam-server binary (runs on your VPS/server)
│
├── SFTP Server (gliderlabs/ssh + pkg/sftp)
├── FTP Server (pyftpdlib equivalent in Go)
├── FTPS Server (FTP over TLS)
│
├── User Manager
│   ├── Local users (beam-server users.yaml)
│   ├── Virtual users (no OS account needed)
│   └── SSH key authentication
│
├── Project Router
│   ├── Maps FTP users to project directories
│   └── Enforces per-project permissions
│
├── Hook Engine (same format as client)
│   ├── on-upload: triggers on incoming files
│   ├── on-connect: triggers on new connection
│   └── on-error: triggers on transfer failure
│
├── MCP Server (exposes server tools to Claude)
│   ├── list_connections
│   ├── kick_user
│   ├── add_user
│   ├── set_permissions
│   └── restart_server
│
└── Admin API (local socket, not exposed to internet)
    └── Used by beam client to manage the server
```

## Beam Client ↔ Beam Server Integration

When a project targets a Beam Server instead of a generic FTP server, the client gains additional capabilities:

```bash
# Standard project (any FTP server)
beam project add mysite --proto sftp --host mysite.com

# Beam Server project (extended management)
beam project add mysite --proto beam --host mysite.com

# Extra commands available for Beam Server projects
beam server status -p mysite        # server health
beam server users -p mysite         # list connected users
beam server logs -p mysite          # server-side logs
beam server restart -p mysite       # restart server process
```

## Server Config Schema

```yaml
# /etc/beam-server/beam-server.yaml

server:
  sftp:
    enabled: true
    port: 22
    host_key: /etc/beam-server/keys/host_rsa

  ftp:
    enabled: false
    port: 21

  ftps:
    enabled: true
    port: 990
    cert: /etc/beam-server/certs/server.crt
    key: /etc/beam-server/certs/server.key

users:
  - name: deploy
    key: ~/.ssh/authorized_keys
    projects:
      - mysite-prod
      - mysite-staging

  - name: client-a
    password: "${CLIENT_A_PASS}"
    projects:
      - client-a-prod

projects:
  - name: mysite-prod
    root: /var/www/mysite
    permissions: rw
    hooks:
      on-upload:
        - type: shell
          cmd: systemctl reload nginx

  - name: client-a-prod
    root: /var/www/clients/client-a
    permissions: rw
    quota: 5GB

mcp:
  enabled: true
  port: 7070
  auth_token: "${BEAM_MCP_TOKEN}"
```

## User Permission Model

```
Server root
├── /var/www/mysite/          ← project: mysite-prod (user: deploy)
│   ├── public_html/
│   └── logs/
├── /var/www/clients/
│   └── client-a/            ← project: client-a-prod (user: client-a)
└── /var/www/staging/         ← project: mysite-staging (user: deploy)
```

Each user is jailed to their assigned project roots. No cross-project access.

## Deployment

```bash
# Install on VPS
curl -sSL https://get.beam.sh/server | bash

# Or as Docker container
docker run -d \
  -p 22:22 -p 990:990 \
  -v /etc/beam-server:/config \
  -v /var/www:/sites \
  beamsh/server:latest

# Systemd service
systemctl enable beam-server
systemctl start beam-server
```

## Compatibility Contract with Client

These interfaces are defined now so the client can be built compatibly:

1. **Config schema**: `projects.yaml` format is identical for client and server
2. **Hook format**: Same YAML structure, same event names
3. **MCP tools**: Server exposes a superset of client MCP tools
4. **Protocol**: Standard FTP/SFTP — Beam client works with non-Beam servers and vice versa

## Implementation Order (Future)

1. Basic SFTP server (single user, single directory)
2. Multi-user with virtual users
3. FTP + FTPS support
4. Hook engine (server-side)
5. MCP server for remote management
6. Admin API + client integration
7. Docker image + install script
8. Web dashboard (optional)
