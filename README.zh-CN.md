# ds-cli

`ds-cli` 是一个通过 REST API 操作已有 Apache DolphinScheduler 集群的单二进制 CLI。它面向 AI agent 和自动化脚本设计：API 命令不交互，并在 stdout 输出一个结构化 JSON envelope，方便稳定解析。

英文文档：[README.md](./README.md)。

## 快速开始

先保存一个命名 DolphinScheduler API profile：

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate
```

创建项目和单任务工作流：

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli task create extract_orders \
  --project-code <project-code> \
  --workflow-name daily_extract_orders \
  --type SHELL \
  --script-file ./extract_orders.sh
ds-cli task online <workflow-code> --project-code <project-code>
```

`ds-cli` 不再安装、启动、停止、配置或升级 DolphinScheduler。它只连接已经运行的 DolphinScheduler API server。

## 安装

### 一键脚本

将最新 release 二进制安装到 `/usr/local/bin`，并默认把内置 skill 安装到 `~/.Codex/skills/` 和 `~/.claude/skills/`。重复执行同一命令即可升级。

```bash
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | bash
```

常用覆盖参数：

```bash
# 固定版本
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | VERSION=v0.1.0 bash

# 安装到用户目录，不使用 sudo
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | PREFIX="$HOME/.local/bin" NO_SUDO=1 bash

# 只安装 skill 到一个目录
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | SKILL_DIR="$HOME/.Codex/skills" bash

# 跳过 skill 安装
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | NO_SKILL=1 bash
```

支持的环境变量：`VERSION`、`PREFIX`、`SKILL_DIR`、`SKILL_DIRS`、`NO_SKILL`、`NO_SUDO`、`REPO`。`SKILL_DIRS` 是冒号分隔的多目录列表，默认值为 `~/.Codex/skills:~/.claude/skills`。

### 源码构建

```bash
make build
bin/ds-cli --help
```

## 配置

API 凭据默认写入 `~/.config/ds-cli/config.yaml`。如需更换配置目录，设置 `DSCLI_CONFIG_DIR`。

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate
ds-cli config cluster list
ds-cli config cluster activate prod
```

配置文件结构：

```yaml
active_cluster: prod
clusters:
  prod:
    api_url: http://ds.example.com/dolphinscheduler
    username: admin
    password: dolphinscheduler123
    timeout: 30s
```

也可以逐次用环境变量覆盖：

```bash
export DSCLI_API_URL=http://localhost:12345/dolphinscheduler
export DSCLI_TOKEN=<access-token>
ds-cli project list
```

解析优先级：

| 配置项 | 优先级 |
|---|---|
| 集群 | `--cluster` -> `DSCLI_CLUSTER` -> `active_cluster` |
| API 地址 | `--api-url` -> `DSCLI_API_URL` -> profile `api_url` |
| 认证 | `--token` / `DSCLI_TOKEN`，然后 `--session-id` / `DSCLI_SESSION_ID`，然后用户名密码 |
| 超时 | `--api-timeout` -> `DSCLI_API_TIMEOUT` -> profile `timeout` -> `30s` |

使用用户名密码时，CLI 会先调用 `/login`，再用返回的 `sessionId` 请求后续接口。

## 命令

| 命令 | 用途 |
|---|---|
| `ds-cli config cluster add/list/activate` | 管理本地命名 DS API profile |
| `ds-cli project create/list/get/delete` | 管理项目 |
| `ds-cli workflow create/update/get/list/online/offline/delete` | 管理工作流定义 |
| `ds-cli task create/online/offline/delete/get/list` | 创建和操作单任务工作流 |
| `ds-cli schedule create/update/get/list/online/offline/delete` | 管理工作流调度 |
| `ds-cli alert group create/update/list/delete` | 管理告警组 |
| `ds-cli environment create/update/list/get/delete` | 管理任务运行环境 |
| `ds-cli --version` | 打印 CLI 版本 |

API 命令组通用 flag：`--cluster`、`--api-url`、`--user`、`--password`、`--token`、`--session-id`、`--api-timeout`。

### 项目

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli project list --page-no 1 --page-size 20
ds-cli project get <project-code>
ds-cli project delete <project-code>
```

### 单任务工作流

`task create` 会创建一个离线工作流定义，里面只有一个 `SHELL` 或 `PYTHON` 任务节点。agent 生成脚本后优先走这条路径。

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

需要直接管理普通工作流定义时，使用 `workflow`：

```bash
ds-cli workflow create daily_job --project-code <project-code>
ds-cli workflow update <workflow-code> --name daily_job_v2
ds-cli workflow list --project-code <project-code>
```

### 调度、告警和环境

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

## 给 AI agent

- 不要等待交互式 prompt。所有输入必须通过 flag、环境变量或文件提供。
- stdout 是 API 结果契约。API/profile 命令只输出一个 JSON envelope。
- stderr 只当诊断信息，不要从 stderr 解析结果。
- 重复操作生产集群时，优先使用命名 cluster profile。
- 单脚本任务优先用 `task create`；只有需要直接操作工作流定义时才用 `workflow`。
- 失败时读取 `ok`、`error.code`、`error.message`、`summary` 和 `data` 中的 DolphinScheduler 原始响应。

## 输出契约

成功 API 命令：

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

命令分发后的配置或 API 失败：

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

## 发布

GitHub Actions 会执行 `go vet`、`gofmt`、race test、构建、`--help` smoke test、安装脚本语法检查和 skill front matter 检查。GoReleaser 打包 `linux/darwin` x `amd64/arm64`，release 包包含二进制、README、LICENSE 和内置 skills。

## 许可

见 [LICENSE](./LICENSE)。
