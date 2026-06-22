---
name: ds
description: 使用 ds-cli 通过 DolphinScheduler REST API 管理已有 DS 集群的项目、单任务工作流、普通工作流、调度、告警组和环境；这是 DolphinScheduler API skill 的短命令别名。
---

# ds-cli API 快捷 Skill

本 skill 使用 `ds-cli` 操作已经运行的 Apache DolphinScheduler API server。`ds-cli` 是非交互式工具：提前准备 profile、flag、环境变量或脚本文件；stdout 只解析 JSON envelope，stderr 只作为诊断信息。

`ds-cli` 不负责安装、配置、启动、停止或升级 DolphinScheduler。

## 工作流

1. 确认用户已有可访问的 DolphinScheduler API 地址，例如 `http://host:12345/dolphinscheduler`。
2. 确认认证方式：优先使用 token，其次 sessionId，再其次用户名密码。
3. 保存命名 API profile，或临时设置环境变量。
4. 根据目标选择命令：
   - 项目：`project create/list/get/delete`
   - 单脚本任务：`task create/online/offline/delete/get/list`
   - 普通工作流：`workflow create/update/get/get-detail/list/online/offline/delete`
   - 改某 task 脚本（一键 offline→update→恢复 release）：`workflow patch-task`
   - 立即触发一次工作流：`workflow start`
   - 工作流实例：`workflow-instance list/get/tasks/control/delete`
   - 任务实例与日志：`task-instance list/log/log-download/force-success/stop`
   - 任务定义（按 task code）：`task-def get/update`
   - 调度：`schedule create/update/get/list/online/offline/delete`
   - 告警组：`alert group create/update/list/delete`
   - 环境：`environment create/update/list/get/delete`

## 配置

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

多集群使用 `--cluster <name>` 指定本次命令，或用 `ds-cli config cluster activate <name>` 设置默认集群。`ds-cli config show` 用来确认当前生效的集群、API 地址、字段来源和认证方式；它只输出 `has_token`、`has_session` 等布尔值，不输出 token、sessionId 或密码明文。

也可逐次覆盖：

```bash
export DSCLI_API_URL=http://localhost:12345/dolphinscheduler
export DSCLI_TOKEN=<access-token>
ds-cli project list
```

优先级：`--cluster` -> `DSCLI_CLUSTER` -> `active_cluster`；认证优先级为 token、sessionId、用户名密码。

## 常用命令

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli project list

ds-cli task create extract \
  --project-code <project-code> \
  --workflow-name daily_extract \
  --type SHELL \
  --script-file ./extract.sh
ds-cli task online <workflow-code> --project-code <project-code>
ds-cli task offline <workflow-code> --project-code <project-code>
ds-cli task delete <workflow-code>

# 读取多任务工作流（含 task 定义 + 关系）
ds-cli workflow get-detail <workflow-code> --project-code <project-code>

# 改一个 task 的 rawScript（自动 offline → 更新 → 恢复 release 状态）
ds-cli workflow patch-task <workflow-code> \
  --project-code <project-code> \
  --task-code <task-code> \
  --raw-script-file ./new.sh

# 立即触发一次工作流
ds-cli workflow start <workflow-code> \
  --project-code <project-code> \
  --environment-code <env-code>

# 查实例、控制实例、查任务、拉日志
ds-cli workflow-instance list --project-code <project-code> --workflow-code <workflow-code>
ds-cli workflow-instance control <instance-id> --project-code <project-code> --type STOP
ds-cli task-instance list --project-code <project-code> --workflow-instance-id <instance-id>
ds-cli task-instance log <task-instance-id> --skip-line-num 0 --limit 500
ds-cli task-instance log-download <task-instance-id> --output ./ti.log

# 按 task code 单独读/写一个任务
ds-cli task-def get <task-code> --project-code <project-code>
ds-cli task-def update <task-code> --project-code <project-code> --raw-script-file ./new.sh

ds-cli schedule create \
  --workflow-code <workflow-code> \
  --crontab "0 0 3 * * ? *" \
  --start-time "2026-01-01 00:00:00" \
  --end-time "2099-01-01 00:00:00" \
  --timezone UTC \
  --environment-code <env-code>

ds-cli alert group create ops --alert-instance-ids 1,2
ds-cli environment create python3 --env-config "export PYTHON_LAUNCHER=/usr/bin/python3"
```

> `schedule create` 现在默认 timezone 为 `UTC`，并要求 `--environment-code` 必填（运行 `ds-cli environment list` 找到现有 code）。`workflow patch-task` 默认会在更新后恢复原 release 状态；传 `--keep-offline` 可保持 OFFLINE。`task-instance log-download` 用 `--output FILE` 落盘并输出 envelope 摘要；不带 `--output` 则把字节流写到 stdout（这是 ds-cli 里唯一一个允许 stdout 非 envelope 的命令）。

## 结果判断

- `ok: true` 表示命令成功。
- `ok: false` 时读取 `error.code`、`error.message` 和 `data`。
- API 命令的 `summary` 包含 `cluster`、`api_url`、`http_status`。
- 不要把 stderr 当成结果数据。
