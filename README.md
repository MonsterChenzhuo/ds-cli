# ds-cli

A single-binary CLI for operating existing Apache DolphinScheduler clusters through the REST API. It is designed for AI agents and automation: API commands are non-interactive and emit one structured JSON envelope on stdout.

Chinese documentation: [README.zh-CN.md](./README.zh-CN.md).

## Quick Start

Save a named DolphinScheduler API profile:

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate
```

Create a project and a single-task workflow:

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli task create extract_orders \
  --project-code <project-code> \
  --workflow-name daily_extract_orders \
  --type SHELL \
  --script-file ./extract_orders.sh
ds-cli task online <workflow-code> --project-code <project-code>
```

`ds-cli` does not install, start, stop, configure, or upgrade DolphinScheduler. Point it at an already running API server.

## Install

### One-Liner

Installs the latest release binary into `/usr/local/bin` and installs bundled skills into `~/.Codex/skills/` and `~/.claude/skills/` by default. Re-run the same command to upgrade.

```bash
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | bash
```

Common overrides:

```bash
# pin a version
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | VERSION=v0.1.0 bash

# install to a non-sudo path
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | PREFIX="$HOME/.local/bin" NO_SUDO=1 bash

# install skills to one custom directory
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | SKILL_DIR="$HOME/.Codex/skills" bash

# skip bundled skills
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | NO_SKILL=1 bash
```

Supported envs: `VERSION`, `PREFIX`, `SKILL_DIR`, `SKILL_DIRS`, `NO_SKILL`, `NO_SUDO`, `REPO`. `SKILL_DIRS` is colon-separated and defaults to `~/.Codex/skills:~/.claude/skills`.

### From Source

```bash
make build
bin/ds-cli --help
```

## Configure

API credentials are stored in `~/.config/ds-cli/config.yaml` by default. Use `DSCLI_CONFIG_DIR` to choose another config directory.

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate
ds-cli config cluster list
ds-cli config cluster activate prod
```

The file shape is:

```yaml
active_cluster: prod
clusters:
  prod:
    api_url: http://ds.example.com/dolphinscheduler
    username: admin
    password: dolphinscheduler123
    timeout: 30s
```

Per-invocation overrides:

```bash
export DSCLI_API_URL=http://localhost:12345/dolphinscheduler
export DSCLI_TOKEN=<access-token>
ds-cli project list
```

Resolution order:

| Setting | Precedence |
|---|---|
| Cluster | `--cluster` -> `DSCLI_CLUSTER` -> `active_cluster` |
| API URL | `--api-url` -> `DSCLI_API_URL` -> profile `api_url` |
| Auth | `--token` / `DSCLI_TOKEN`, then `--session-id` / `DSCLI_SESSION_ID`, then username/password |
| Timeout | `--api-timeout` -> `DSCLI_API_TIMEOUT` -> profile `timeout` -> `30s` |

Password login calls `/login` first and then reuses the returned `sessionId`.

## Commands

| Command | Purpose |
|---|---|
| `ds-cli config cluster add/list/activate` | Manage named DolphinScheduler API profiles |
| `ds-cli project create/list/get/delete` | Manage projects |
| `ds-cli workflow create/update/get/list/online/offline/delete` | Manage workflow definitions |
| `ds-cli task create/online/offline/delete/get/list` | Create and operate single-task workflows |
| `ds-cli schedule create/update/get/list/online/offline/delete` | Manage workflow schedules |
| `ds-cli alert group create/update/list/delete` | Manage alert groups |
| `ds-cli environment create/update/list/get/delete` | Manage task runtime environments |
| `ds-cli --version` | Print the CLI version |

Common API flags are available on API command groups: `--cluster`, `--api-url`, `--user`, `--password`, `--token`, `--session-id`, and `--api-timeout`.

### Projects

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli project list --page-no 1 --page-size 20
ds-cli project get <project-code>
ds-cli project delete <project-code>
```

### Single-Task Workflows

`task create` creates an offline workflow definition containing one `SHELL` or `PYTHON` task node. It is the agent-friendly path for script jobs.

```bash
ds-cli task create extract_orders \
  --project-code <project-code> \
  --workflow-name daily_extract_orders \
  --type SHELL \
  --script-file ./extract_orders.sh \
  --worker-group default

ds-cli task online <workflow-code> --project-code <project-code>
ds-cli task offline <workflow-code> --project-code <project-code>
ds-cli task delete <workflow-code>
```

Use `workflow` when the caller needs direct workflow-definition operations:

```bash
ds-cli workflow create daily_job --project-code <project-code>
ds-cli workflow update <workflow-code> --name daily_job_v2
ds-cli workflow list --project-code <project-code>
```

### Schedules, Alerts, and Environments

```bash
ds-cli schedule create \
  --workflow-code <workflow-code> \
  --crontab "0 0 2 * * ? *" \
  --start-time "2026-01-01 00:00:00" \
  --end-time "2099-01-01 00:00:00" \
  --timezone Asia/Shanghai \
  --warning-type FAILURE \
  --warning-group-id <alert-group-id>

ds-cli alert group create ops --alert-instance-ids 1,2
ds-cli environment create python3 \
  --env-config "export PYTHON_LAUNCHER=/usr/bin/python3" \
  --worker-groups default
```

## For AI Agents

- Do not expect prompts. Provide every input through flags, environment variables, or files.
- Treat stdout as the API result contract. API/profile commands write one JSON envelope.
- Treat stderr as diagnostics only. Do not parse stderr as result data.
- Prefer named cluster profiles for repeated operations.
- Prefer `task create` for a one-script job. Use `workflow` only for direct workflow-definition management.
- On failure, inspect `ok`, `error.code`, `error.message`, `summary`, and the DolphinScheduler response under `data`.

## Output Contract

Successful API command:

```json
{
  "command": "project.list",
  "ok": true,
  "summary": {
    "cluster": "prod",
    "api_url": "http://ds.example.com/dolphinscheduler",
    "http_status": 200
  },
  "data": {
    "code": 0,
    "msg": "success",
    "data": []
  }
}
```

Configuration or API failure after command dispatch:

```json
{
  "command": "project.list",
  "ok": false,
  "error": {
    "code": "CONFIG_ERROR",
    "message": "api_url is required: configure ds-cli config cluster add <name> --api-url, set DSCLI_API_URL, or pass --api-url"
  }
}
```

## Release

GitHub Actions run `go vet`, `gofmt`, race tests, build, `--help`, installer syntax checks, and skill front matter checks. GoReleaser packages `linux/darwin` x `amd64/arm64` archives containing the binary, README files, LICENSE, and bundled skills.

## License

See [LICENSE](./LICENSE).
