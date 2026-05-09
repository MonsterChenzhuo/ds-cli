---
name: dolphinscheduler-pseudo-cluster
description: 使用 ds-cli 在本机直接部署 Apache DolphinScheduler 3.4.1 伪集群；适用于“部署 dolphinscheduler 伪集群”“安装 DS 3.4.1”“初始化 DolphinScheduler MySQL 元数据库”等请求。
---

# DolphinScheduler 3.4.1 伪集群部署

本 skill 使用 `ds-cli` 在当前机器直接部署 DolphinScheduler 3.4.1。不要生成 SSH inventory；v1 只支持本机部署。

## 工作流

1. 确认用户提供 MySQL 连接信息：host、port、database、username、password。
2. 如果用户希望 CLI 创建数据库，还需要 MySQL 管理员账号，并在配置中设置 `mysql.create_database: true`。
3. 写入 `ds.yaml`，可从 `ds.yaml.example` 复制。
4. 按顺序执行：

```bash
ds-cli preflight
ds-cli install
ds-cli configure
ds-cli init-db
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

## 默认登录

访问：

```text
http://localhost:12345/dolphinscheduler/ui
```

账号：

```text
admin / dolphinscheduler123
```

## 注意

- DolphinScheduler 3.4.1 的二进制包不包含插件依赖；`ds-cli` 会放置 MySQL JDBC Driver 用于元数据库初始化。
- 运行 shell、Spark、Hive、Flink 等任务所需的外部运行时不在 v1 部署范围内。
