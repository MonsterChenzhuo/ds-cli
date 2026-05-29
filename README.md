# ds-cli

An AI-first, single-binary Go CLI for Codex and Claude Code to deploy and operate Apache DolphinScheduler 3.4.1. The CLI is intentionally non-interactive: agents pass config files, flags, or environment variables, then parse one JSON envelope from stdout. Human-readable progress goes to stderr.

Chinese documentation: [README.zh-CN.md](./README.zh-CN.md).

## Quick Start

Deploy a local pseudo-cluster:

```bash
cp ds.yaml.example ds.yaml
$EDITOR ds.yaml
ds-cli bootstrap
ds-cli status
```

After the API server is running, save a named DolphinScheduler API cluster profile:

```bash
ds-cli config cluster add local \
  --api-url http://localhost:12345/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate
```

Create and publish a single-task workflow for an agent-generated shell script:

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli task create extract_orders \
  --project-code <project-code> \
  --workflow-name daily_extract_orders \
  --type SHELL \
  --script-file ./extract_orders.sh
ds-cli task online <workflow-code> --project-code <project-code>
```

## Install

### One-Liner

Installs the latest release binary into `/usr/local/bin` and installs bundled skills into both `~/.Codex/skills/` and `~/.claude/skills/` by default. Re-run the same command to upgrade.

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

## Configure Deployment

`ds-cli` deploys either a local pseudo-cluster or a distributed cluster. Config lookup order:

```text
--config <path> -> $DSCLI_CONFIG -> ./ds.yaml -> ~/.ds-cli/ds.yaml
```

Pseudo-cluster:

```bash
cp ds.yaml.example ds.yaml
```

Distributed cluster:

```bash
cp ds.distributed.yaml.example ds.yaml
```

Core distributed fields:

```yaml
cluster:
  mode: distributed

ssh:
  user: dolphinscheduler
  private_key: ~/.ssh/id_rsa
  port: 22
  parallelism: 4

hosts:
  - { name: ds1, address: 10.0.0.1 }
  - { name: ds2, address: 10.0.0.2 }
  - { name: ds3, address: 10.0.0.3 }

roles:
  zookeeper: [ds1, ds2, ds3]
  api_server: [ds1]
  master_server: [ds1, ds2]
  worker_server: [ds2, ds3]
  alert_server: [ds1]
```

Reuse an external ZooKeeper instead of letting `ds-cli` manage it:

```yaml
zookeeper:
  external_connect_string: zk1:2181,zk2:2181,zk3:2181
```

Runtime environment variables are rendered into `dolphinscheduler_env.sh`:

```yaml
env:
  python_launcher: /usr/bin/python3
  hadoop_user_name: airflow
  java_home: /data/hadoopclient/JDK/jdk1.8.0_272
  hadoop_home: /data/hadoopclient/HDFS/hadoop
  path_prepend:
    - $HADOOP_HOME/bin
    - $HADOOP_HOME/sbin
  exports:
    SPARK_HOME: /data/spark
    HIVE_HOME: /data/hive
```

`cluster.java_home` controls the JDK used or installed by `ds-cli`; `env.java_home` controls DolphinScheduler service runtime `JAVA_HOME`.

## Deployment Commands

```bash
ds-cli preflight
ds-cli install
ds-cli configure
ds-cli init-db
ds-cli plugins --restart
ds-cli start
ds-cli status
```

Or run the full lifecycle:

```bash
ds-cli bootstrap
```

`install` and `bootstrap` install default task plugins through DolphinScheduler's official flow: render `conf/plugins_config`, run `bash ./bin/install-plugins.sh 3.4.1`, and verify the configured jars under `plugins/task-plugins/`.

## API Cluster Profiles

REST API credentials are stored separately from deployment config at `~/.config/ds-cli/config.yaml`.

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate

ds-cli config cluster list
ds-cli config cluster activate prod
```

Per-invocation overrides:

```bash
export DSCLI_API_URL=http://localhost:12345/dolphinscheduler
export DSCLI_TOKEN=<access-token>
ds-cli project list
```

Authentication precedence: `--token` / `DSCLI_TOKEN`, then `--session-id` / `DSCLI_SESSION_ID`, then `--user --password` / `DSCLI_USER DSCLI_PASSWORD`. Password login calls `/login` first and then reuses the returned `sessionId`.

## API Commands

| Command | Purpose |
|---|---|
| `ds-cli config cluster add/list/activate` | Manage named DS API cluster profiles |
| `ds-cli project create/list/get/delete` | Manage DolphinScheduler projects |
| `ds-cli workflow create/update/get/list/online/offline/delete` | Manage workflow definitions |
| `ds-cli task create/online/offline/delete/get/list` | Agent-friendly single-task workflow helpers |
| `ds-cli schedule create/update/get/list/online/offline/delete` | Manage workflow schedules |
| `ds-cli alert group create/update/list/delete` | Manage alert groups |
| `ds-cli environment create/update/list/get/delete` | Manage DolphinScheduler environments |

Schedule and alert example:

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

This CLI is optimized for Codex and Claude Code, not for manual terminal workflows.

- Do not expect prompts. Provide every input through YAML, flags, env vars, or files.
- Treat stdout as the contract. It is always a JSON envelope for operational commands.
- Treat stderr as progress and diagnostics. Do not parse stderr as result data.
- Prefer named API cluster profiles before creating projects, tasks, schedules, alerts, or environments.
- Prefer `task create` for single-script jobs generated by an agent; use `workflow` only when the caller needs direct workflow-definition operations.
- On failure, inspect `ok`, `error`, `steps[].ok`, `steps[].message`, and the run log under `~/.ds-cli/runs/<run-id>/`.

## Output Contract

Deployment command envelope:

```json
{
  "command": "bootstrap",
  "ok": true,
  "steps": [
    {
      "name": "preflight",
      "ok": true,
      "elapsed_ms": 42
    }
  ],
  "run_id": "20260530120000-bootstrap",
  "config_path": "/path/to/ds.yaml"
}
```

API command envelope:

```json
{
  "command": "project.list",
  "ok": true,
  "summary": {
    "cluster": "local",
    "api_url": "http://localhost:12345/dolphinscheduler",
    "http_status": 200
  },
  "data": {
    "code": 0,
    "msg": "success",
    "data": []
  }
}
```

Failures keep the same envelope shape with `ok: false` plus either `error` or failed `steps[]`.

## Operations

```bash
ds-cli stop
ds-cli start
ds-cli restart worker
ds-cli restart api worker
ds-cli restart zookeeper
ds-cli restart all
ds-cli status
ds-cli plugins --restart
ds-cli systemd
ds-cli uninstall
ds-cli uninstall --purge-data
```

`restart` supports `api`/`api-server`, `master`/`master-server`, `worker`/`worker-server`, `alert`/`alert-server`, `zk`/`zookeeper`, and `all`. Distributed mode targets hosts by `roles`. If `zookeeper.external_connect_string` is set, `ds-cli restart zookeeper` fails instead of touching an external cluster.

`systemd` installs service units with `Restart=on-failure`.

## Release

GitHub Actions run vet, gofmt, race tests, build, help smoke checks, and skill front matter checks. GoReleaser packages `linux/darwin` x `amd64/arm64` archives containing the binary, `README.md`, `README.zh-CN.md`, examples, and bundled skills.

## License

See [LICENSE](./LICENSE).
