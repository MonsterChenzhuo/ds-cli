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
ds-cli config init
ds-cli config cluster add prod \
  --api-url https://dolphinscheduler.example.com/dolphinscheduler \
  --token '<access-token>' \
  --activate
ds-cli config cluster add staging \
  --api-url https://staging-ds.example.com/dolphinscheduler \
  --token '<access-token>'
ds-cli config cluster list
ds-cli config show
ds-cli config cluster activate prod
```

配置文件结构：

```yaml
active_cluster: prod
clusters:
  prod:
    api_url: https://dolphinscheduler.example.com/dolphinscheduler
    token: <access-token>
    timeout: 30s
  staging:
    api_url: https://staging-ds.example.com/dolphinscheduler
    token: <access-token>
    timeout: 30s
```

`ds-cli config show` 会输出当前生效的 profile、字段来源和认证方式，但只显示 `has_token`、`has_session` 等布尔字段，不输出 token、sessionId 或密码明文。配置文件权限写为 `0600`。

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
| `ds-cli config init/show` | 初始化配置模板、查看当前生效配置 |
| `ds-cli config cluster add/list/activate/show` | 管理本地命名 DS API profile（`show --reveal-token` / `--shell` 便于脚本集成） |
| `ds-cli project create/list/get/delete` | 管理项目 |
| `ds-cli workflow create/update/get/get-detail/list/online/offline/delete` | 管理工作流定义；`get-detail` 一次返回工作流 + 全部 task + 关系 |
| `ds-cli workflow patch-task` | 替换多任务工作流中某个 task 的 `rawScript`（自动 offline → 更新 → 恢复 release 状态） |
| `ds-cli workflow start` | 通过 `/executors/start-workflow-instance` 立即触发一次工作流 |
| `ds-cli workflow-instance list/get/tasks/control/delete` | 查询和控制工作流实例（`control --type STOP\|PAUSE\|RESUME\|RERUN\|RECOVER-FAILED`） |
| `ds-cli task-instance list/log/log-download/force-success/stop` | 查询任务实例并拉取 worker 日志 |
| `ds-cli task-def get/update` | 按 code 单独读取或更新一个任务定义（使用 `with-upstream` 接口） |
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

# 读取一个工作流及其所有 task 定义和上下游关系：
ds-cli workflow get-detail <workflow-code> --project-code <project-code>

# 原地替换某个 task 的 rawScript（自动 offline / 更新 / 恢复 release 状态）：
ds-cli workflow patch-task <workflow-code> \
  --project-code <project-code> \
  --task-code <task-code> \
  --raw-script-file ./new_check_partition.sh

# 立即触发一次工作流：
ds-cli workflow start <workflow-code> \
  --project-code <project-code> \
  --environment-code <env-code>
```

### 工作流实例与任务实例

```bash
ds-cli workflow-instance list --project-code <project-code> --workflow-code <workflow-code>
ds-cli workflow-instance get <instance-id> --project-code <project-code>
ds-cli workflow-instance tasks <instance-id> --project-code <project-code>
ds-cli workflow-instance control <instance-id> --project-code <project-code> --type STOP

ds-cli task-instance list --project-code <project-code> --workflow-instance-id <instance-id>
ds-cli task-instance log <task-instance-id> --skip-line-num 0 --limit 500
ds-cli task-instance log-download <task-instance-id> --output ./ti.log
ds-cli task-instance force-success <task-instance-id> --project-code <project-code>
```

### 按 code 单独管理 task

```bash
ds-cli task-def get <task-code> --project-code <project-code>
ds-cli task-def update <task-code> --project-code <project-code> --raw-script-file ./new.sh
```

### 调度、告警和环境

```bash
ds-cli schedule create \
  --workflow-code <workflow-code> \
  --crontab "0 0 2 * * ? *" \
  --start-time "2026-01-01 00:00:00" \
  --end-time "2099-01-01 00:00:00" \
  --timezone UTC \
  --environment-code <env-code> \
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
