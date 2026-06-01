---
name: dolphinscheduler-pseudo-cluster
description: 使用 ds-cli 通过 DolphinScheduler REST API 管理已有 DS 集群的项目、单任务工作流、普通工作流、调度、告警组和环境；保留此 skill 名称用于兼容旧安装路径。
---

# DolphinScheduler API 操作

本 skill 使用 `ds-cli` 操作已经运行的 Apache DolphinScheduler API server。当前 `ds-cli` 是纯 REST API 客户端，不再提供 DolphinScheduler 安装、伪集群、分布式部署、ZooKeeper、MySQL 初始化、插件安装或 systemd 管理能力。

## 前置条件

- 用户已经有可访问的 DolphinScheduler API 地址，例如 `http://host:12345/dolphinscheduler`。
- 用户提供 token、sessionId，或用户名密码。
- 如果要创建任务，目标 DolphinScheduler 集群本身需要已经具备对应 worker group、tenant、environment 和 task plugin。

## 配置命名 API 集群

```bash
ds-cli config cluster add prod \
  --api-url http://ds.example.com/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate

ds-cli config cluster list
ds-cli config cluster activate prod
```

临时调用也可以使用：

```bash
export DSCLI_API_URL=http://localhost:12345/dolphinscheduler
export DSCLI_TOKEN=<access-token>
ds-cli project list
```

解析优先级：

- 集群：`--cluster` -> `DSCLI_CLUSTER` -> `active_cluster`
- API 地址：`--api-url` -> `DSCLI_API_URL` -> profile `api_url`
- 认证：token -> sessionId -> username/password
- 超时：`--api-timeout` -> `DSCLI_API_TIMEOUT` -> profile `timeout` -> `30s`

## 标准操作流程

1. 先执行 `ds-cli project list` 确认可连通。
2. 需要创建项目时执行 `ds-cli project create <name>`。
3. agent 生成单脚本任务时，优先执行 `ds-cli task create`，再根据需要 `task online`。
4. 已有复杂工作流时，使用 `workflow` 命令直接管理工作流定义。
5. 需要定时运行时，使用 `schedule create` 后 `schedule online`。
6. 告警组使用 `alert group`，运行环境使用 `environment`。

## 常用命令

```bash
ds-cli project create demo --description "created by ds-cli"

ds-cli task create extract \
  --project-code <project-code> \
  --workflow-name daily_extract \
  --type SHELL \
  --script-file ./extract.sh
ds-cli task online <workflow-code> --project-code <project-code>
ds-cli task offline <workflow-code> --project-code <project-code>
ds-cli task delete <workflow-code>

ds-cli workflow create daily_job --project-code <project-code>
ds-cli workflow list --project-code <project-code>

ds-cli schedule create \
  --workflow-code <workflow-code> \
  --crontab "0 0 2 * * ? *" \
  --start-time "2026-01-01 00:00:00" \
  --end-time "2099-01-01 00:00:00"
ds-cli schedule online <schedule-id> --project-code <project-code>

ds-cli alert group create ops --alert-instance-ids 1,2
ds-cli environment create python3 --env-config "export PYTHON_LAUNCHER=/usr/bin/python3"
```

## 输出契约

stdout 只解析 JSON envelope：

- `ok: true` 表示命令成功。
- `ok: false` 时读取 `error.code`、`error.message`。
- `summary` 包含 `cluster`、`api_url`、`http_status`。
- `data` 是 DolphinScheduler API 的原始响应体。
- stderr 只作为诊断信息，不承载结果数据。
