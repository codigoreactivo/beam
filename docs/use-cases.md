# Beam — Use Cases

## UC-01: Developer deploys a static site build

**Interface**: CLI  
**Trigger**: Manual after `npm run build`

```bash
beam deploy -p mysite-prod
# → runs pre-deploy hook (optional build step)
# → syncs ./dist to /public_html (incremental)
# → runs post-deploy hook (cache invalidation, notify)
```

---

## UC-02: Developer watches files during active development

**Interface**: CLI / TUI  
**Trigger**: Continuous while working

```bash
beam watch -p mysite-staging
# → monitors ./src for changes
# → debounces 500ms
# → uploads changed files automatically
# → shows live feedback in terminal
```

---

## UC-03: Designer uploads assets without touching a terminal

**Interface**: GUI  
**Trigger**: Manual drag and drop

```
Open Beam GUI → select project → drag images into remote panel → done
```

---

## UC-04: Claude deploys to production via MCP

**Interface**: MCP  
**Trigger**: Natural language in Claude chat

```
User: "Beam, deploy mysite to production and clear the cache"

Claude (via MCP tools):
  1. beam_sync({ project: "mysite-prod", dryRun: true })
  2. Shows diff to user for confirmation
  3. beam_deploy({ project: "mysite-prod" })
  4. beam_hook_run({ project: "mysite-prod", hook: "clear-cache" })
```

---

## UC-05: Developer manages multiple client sites simultaneously

**Interface**: TUI  
**Trigger**: Manual, switching between clients

```
beam tui
→ Sidebar lists all projects
→ Tab switches between client-a, client-b, client-c
→ Each project shows its own file diff and activity log
→ Upload to each independently without opening multiple windows
```

---

## UC-06: CI/CD pipeline deploys on git push

**Interface**: CLI (non-interactive, JSON output)  
**Trigger**: GitHub Actions / GitLab CI / any CI runner

```yaml
# .github/workflows/deploy.yml
- name: Deploy to production
  run: |
    beam deploy -p mysite-prod --quiet --json
  env:
    BEAM_CONFIG: ${{ secrets.BEAM_CONFIG }}
```

---

## UC-07: Developer checks what's different before deploying

**Interface**: CLI  
**Trigger**: Manual sanity check

```bash
beam diff -p mysite-prod
# Output:
# + contact.html    (local only, will be uploaded)
# ~ index.html      (modified locally)
# - old-page.html   (remote only, will not be touched)
```

---

## UC-08: Claude audits all projects and reports status

**Interface**: MCP  
**Trigger**: Natural language in Claude chat

```
User: "Which of my projects haven't been deployed in the last 7 days?"

Claude (via MCP):
  1. beam_project_list() → gets all projects
  2. beam_project_logs({ project: each, since: "7d" })
  3. Analyzes last deploy timestamps
  4. Returns summary report
```

---

## UC-09: Developer recovers from a bad deploy

**Interface**: CLI / TUI / MCP  
**Trigger**: Something broke after upload

```bash
# CLI
beam logs -p mysite-prod --last 20
beam download /public_html/index.html ./backup.html -p mysite-prod
beam upload ./previous-build/index.html /public_html/ -p mysite-prod

# Or via Claude MCP
User: "Rollback mysite-prod to the last known good version"
```

---

## UC-10: New team member sets up all projects from shared config

**Interface**: CLI  
**Trigger**: Onboarding

```bash
# Config file shared via git repo or secret manager
beam import ./team-projects.yaml
beam project test --all
# → tests all connections, reports which are reachable
```

---

## UC-11: Developer sets up post-deploy webhook to Slack

**Interface**: CLI (config edit) or GUI (settings panel)  
**Trigger**: One-time setup

```bash
beam project edit mysite-prod
# Opens project config, user adds hook:
# post-deploy:
#   - type: webhook
#     url: https://hooks.slack.com/...
#     body: '{"text": "mysite deployed successfully"}'
```

---

## UC-12: File watcher paused during a large refactor

**Interface**: TUI  
**Trigger**: Developer temporarily stops auto-sync

```
In TUI → press W → watcher pauses (shows ⏸ in status bar)
→ developer reorganizes files locally
→ press W again → watcher resumes
→ only net-changed files are uploaded
```

---

## UC-13: Quick file inspection without a full client

**Interface**: CLI  
**Trigger**: Ad-hoc remote exploration

```bash
beam ls /public_html -p mysite-prod
beam ls /public_html/assets -p mysite-prod --long
beam download /public_html/config.php ./local-config.php -p mysite-prod
```

---

## UC-14: Claude suggests optimizations based on deploy patterns

**Interface**: MCP  
**Trigger**: Developer asks for insight

```
User: "Am I deploying efficiently? Look at my last 30 deploys on mysite"

Claude (via MCP):
  → Reads transfer logs
  → Identifies: "You're uploading node_modules 12 times — add it to .beamignore"
  → Identifies: "Your pre-deploy hook takes 45s — consider caching your build"
```

---

## UC-15: Server management (Future — Beam Server)

**Interface**: CLI + MCP  
**Trigger**: When running beam-server on a VPS

```bash
beam server status -p my-vps
beam server users -p my-vps
beam server logs -p my-vps --tail 50

# Via MCP
User: "Who is connected to my server right now?"
Claude: beam_server_connections({ project: "my-vps" }) → lists active sessions
```
