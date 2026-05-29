---
name: dolphinscheduler-pseudo-cluster
description: 使用 ds-cli 部署 Apache DolphinScheduler 3.4.1，支持本机伪集群和多机分布式；适用于“部署 dolphinscheduler”“安装 DS 3.4.1”“初始化 DolphinScheduler MySQL 元数据库”等请求。
---

# DolphinScheduler 3.4.1 部署

本 skill 使用 `ds-cli` 部署 DolphinScheduler 3.4.1。默认本机伪集群；用户要求“完整分布式/多机/集群部署”时，生成包含 `hosts`、`ssh`、`roles` 的 `ds.yaml`。

## 工作流

1. 确认用户提供 MySQL 连接信息：host、port、database、username、password。
2. 如果用户希望 CLI 创建数据库，还需要 MySQL 管理员账号，并在配置中设置 `mysql.create_database: true`。
3. 写入 `ds.yaml`，可从 `ds.yaml.example` 复制。
   - 伪集群：使用 `cluster.mode: pseudo`。
   - 分布式：使用 `cluster.mode: distributed`，填写 `hosts`、`ssh`、`roles`。
   - 复用外部 ZooKeeper：填写 `zookeeper.external_connect_string`，不要要求 `roles.zookeeper`。
   - 由 ds-cli 安装 ZooKeeper：填写奇数个 `roles.zookeeper`。
   - 默认 task 插件为 `shell` 和 `python`，如需显式配置：`plugins.task: [shell, python]`。
   - `ds-cli` 会写入 `conf/plugins_config` 并执行官方 `bash ./bin/install-plugins.sh 3.4.1` 安装插件。
   - 如需 Python/Hadoop/Java 运行时环境，使用 `env.python_launcher`、`env.hadoop_user_name`、`env.java_home`、`env.hadoop_home`、`env.path_prepend` 或 `env.exports`。
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
- 需要按组件重启时，使用 `ds-cli restart worker`、`ds-cli restart api worker`、`ds-cli restart zookeeper` 或 `ds-cli restart all`；分布式模式会根据 `roles` 自动定位主机。

## 默认登录

访问：

```text
http://localhost:12345/dolphinscheduler/ui
```

账号：

```text
admin / dolphinscheduler123
```

## API 管理

部署完成后可配置命名 API 集群，后续命令会通过 DolphinScheduler REST API 操作项目、任务、调度、告警和环境：

```bash
ds-cli config cluster add local \
  --api-url http://localhost:12345/dolphinscheduler \
  --user admin \
  --password dolphinscheduler123 \
  --activate
```

常用命令：

```bash
ds-cli project create demo --description "created by ds-cli"
ds-cli task create extract --project-code <project-code> --workflow-name daily_extract --script-file ./extract.sh
ds-cli task online <workflow-code> --project-code <project-code>
ds-cli task offline <workflow-code> --project-code <project-code>
ds-cli task delete <workflow-code>
ds-cli schedule create --workflow-code <workflow-code> --crontab "0 0 2 * * ? *" --start-time "2026-01-01 00:00:00" --end-time "2099-01-01 00:00:00"
ds-cli alert group create ops --alert-instance-ids 1,2
ds-cli environment create python3 --env-config "export PYTHON_LAUNCHER=/usr/bin/python3"
```

也可用 `DSCLI_API_URL`、`DSCLI_TOKEN`、`DSCLI_SESSION_ID`、`DSCLI_USER`、`DSCLI_PASSWORD` 临时覆盖 profile。新增 API 命令 stdout 输出 JSON envelope。

## 注意

- DolphinScheduler 3.4.1 的二进制包不包含插件依赖；`ds-cli` 会使用官方 `install-plugins.sh` 安装 `shell`、`python` task 插件，并放置 MySQL JDBC Driver 用于元数据库初始化。
- 运行 shell、Spark、Hive、Flink 等任务所需的外部运行时不在 v1 部署范围内。
- 分布式模式通过 SSH 执行，目标机器需要 SSH 可达，且 `ssh.user` 具备安装目录、数据目录和 Java 安装所需权限。
