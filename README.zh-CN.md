# ds-cli

`ds-cli` 是面向 Claude Code 使用的单二进制 Go CLI，用于在当前机器直接部署 Apache DolphinScheduler 3.4.1 伪集群。它会安装或复用 JDK 11、安装 ZooKeeper、下载 DolphinScheduler 二进制包和 MySQL JDBC Driver，渲染 MySQL 元数据库配置，并执行数据库初始化与启停管理。

## 范围

- 部署模式：单机伪集群，`api-server`、`master-server`、`worker-server`、`alert-server` 都在本机。
- DolphinScheduler：固定支持 `3.4.1`。
- 注册中心：本机 ZooKeeper，默认 `localhost:2181`。
- 元数据库：使用用户提供的 MySQL。`ds-cli` 默认假设数据库和用户已存在；如果 `mysql.create_database: true`，会使用管理员账号通过本机 `mysql` CLI 创建数据库。
- Java：如果 `cluster.java_home` 不存在，会优先复用系统 JDK 11，否则尝试通过 `apt-get`、`dnf`、`yum` 或 `brew` 安装 OpenJDK 11。

## 安装

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | bash
```

脚本会自动识别 `linux/darwin` 和 `amd64/arm64`，下载 GitHub Release 中对应的 `ds-cli` 压缩包，校验 checksum，安装二进制，并把内置 Claude Code skill 安装到 `~/.ds-cli/skills/`。

常用覆盖参数：

```bash
# 固定版本
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | VERSION=v0.1.0 bash

# 安装到用户目录，不使用 sudo
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | PREFIX=$HOME/.local/bin NO_SUDO=1 bash

# 私有 fork 或仓库名变化时
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | REPO=your-org/ds-cli bash
```

### 从源码构建

```bash
make build
bin/ds-cli --help
```

### GitHub 自动打包

仓库内置 GitHub Actions：

- `.github/workflows/ci.yml`：push/PR 时执行 `go vet`、`gofmt`、`go test -race`、构建和 skill front matter 检查。
- `.github/workflows/release.yml`：推送到 `main` 时自动递增 patch tag；推送 `v*` tag 时由 GoReleaser 打包 `linux/darwin`、`amd64/arm64` release artifact。

GoReleaser 会把 `README`、`LICENSE`、`ds.yaml.example` 和 `skills/**/*` 一起放入 tar.gz 包，供一键安装脚本安装。

## 配置

复制示例配置：

```bash
cp ds.yaml.example ds.yaml
```

修改 MySQL 连接信息：

```yaml
mysql:
  host: 127.0.0.1
  port: 3306
  database: dolphinscheduler
  username: ds_user
  password: ds_password
```

配置查找顺序为：`--config <path>` -> `$DSCLI_CONFIG` -> `./ds.yaml` -> `~/.ds-cli/ds.yaml`。

## 部署

```bash
ds-cli preflight
ds-cli install
ds-cli configure
ds-cli init-db
ds-cli start
ds-cli status
```

也可以一条命令完成：

```bash
ds-cli bootstrap
```

每条命令 stdout 都输出 JSON envelope，stderr 输出进度。详细 stdout/stderr 日志写入 `~/.ds-cli/runs/<run-id>/`。

## 登录

默认访问：

```text
http://localhost:12345/dolphinscheduler/ui
```

默认账号密码：

```text
admin / dolphinscheduler123
```

## 运维

```bash
ds-cli stop
ds-cli start
ds-cli status
ds-cli uninstall
ds-cli uninstall --purge-data
```

`--purge-data` 会删除 `cluster.data_dir`，谨慎使用。
