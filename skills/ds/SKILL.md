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
   - 普通工作流：`workflow create/update/get/list/online/offline/delete`
   - 调度：`schedule create/update/get/list/online/offline/delete`
   - 告警组：`alert group create/update/list/delete`
   - 环境：`environment create/update/list/get/delete`

## 配置

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate

ds-cli config cluster list
ds-cli config cluster activate prod
```

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

ds-cli schedule create \
  --workflow-code <workflow-code> \
  --crontab "0 0 2 * * ? *" \
  --start-time "2026-01-01 00:00:00" \
  --end-time "2099-01-01 00:00:00"

ds-cli alert group create ops --alert-instance-ids 1,2
ds-cli environment create python3 --env-config "export PYTHON_LAUNCHER=/usr/bin/python3"
```

## 结果判断

- `ok: true` 表示命令成功。
- `ok: false` 时读取 `error.code`、`error.message` 和 `data`。
- API 命令的 `summary` 包含 `cluster`、`api_url`、`http_status`。
- 不要把 stderr 当成结果数据。
