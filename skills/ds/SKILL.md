---
name: ds
description: 使用 ds-cli 部署 Apache DolphinScheduler 3.4.1，支持本机伪集群和多机分布式；这是 dolphinscheduler-pseudo-cluster skill 的短命令别名，适合在 Claude Code 中输入 /ds 调用。
---

# ds-cli 快捷 Skill

这是 `dolphinscheduler-pseudo-cluster` 的短别名。使用本 skill 时，按以下流程驱动 `ds-cli` 部署 DolphinScheduler 3.4.1。默认伪集群；用户要求完整分布式时生成分布式 `ds.yaml`。

## 工作流

1. 确认用户提供 MySQL 连接信息：host、port、database、username、password。
2. 如果用户希望 CLI 创建数据库，还需要 MySQL 管理员账号，并在配置中设置 `mysql.create_database: true`。
3. 写入 `ds.yaml`，可从 `ds.yaml.example` 复制。
   - 默认 task 插件为 `shell` 和 `python`，如需显式配置：`plugins.task: [shell, python]`。
4. 按顺序执行：

```bash
ds-cli preflight
ds-cli install
ds-cli configure
ds-cli init-db
ds-cli plugins --restart
ds-cli start
ds-cli status
```

或者首次部署直接执行：

```bash
ds-cli bootstrap
```

## 结果判断

每条命令 stdout 会输出 JSON envelope：

- `ok: true` 表示命令成功。
- `steps[].ok: false` 表示某个步骤失败。
- 失败时读取 `~/.ds-cli/runs/<run-id>/<step>.stderr`。
- `status` 会逐服务核对进程，worker 缺失时必须视为失败。

## 默认登录

访问：

```text
http://localhost:12345/dolphinscheduler/ui
```

账号：

```text
admin / dolphinscheduler123
```
