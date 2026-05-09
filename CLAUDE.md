# CLAUDE.md

本仓库是 `ds-cli`：一个面向 Claude Code 使用的单二进制 Go CLI，用于在当前机器直接部署 Apache DolphinScheduler 3.4.1 伪集群。

## 项目范围

- v1 只支持本机直接部署，不做 SSH inventory，也不复用 `hadoop-cli` 的远程 orchestrator。
- 部署目标是 DolphinScheduler 3.4.1 伪集群：`api-server`、`master-server`、`worker-server`、`alert-server` 都在本机。
- 注册中心使用本机 ZooKeeper。
- 元数据库使用用户提供的 MySQL；默认假设库和账号已存在，只有配置 `mysql.create_database: true` 时才用管理员账号创建数据库。
- CLI 会安装或复用 JDK 11、安装 ZooKeeper、下载 DolphinScheduler 二进制包、下载 MySQL JDBC Driver、渲染配置并执行库表初始化。

## 常用命令

```bash
make build        # go build -o bin/ds-cli .
make test         # go test ./...
make fmt          # gofmt -w main.go cmd internal
make tidy         # go mod tidy
make all          # fmt + tidy + test + build
```

发布前至少执行：

```bash
bash -n scripts/install.sh
go vet ./...
test -z "$(gofmt -l .)"
go test ./... -race
go build -o bin/ds-cli .
./bin/ds-cli --help
```

## 一键安装与发布

线上安装命令：

```bash
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | bash
```

安装脚本约定：

- 自动识别 `linux/darwin` 和 `amd64/arm64`。
- 默认从 GitHub latest release 下载 `ds-cli_<version>_<os>_<arch>.tar.gz`。
- 下载 `ds-cli_<version>_checksums.txt` 并校验 checksum。
- 安装二进制到 `/usr/local/bin/ds-cli`。
- 将 release 包内的 `skills/` 安装到 `~/.ds-cli/skills/`。
- 将 `ds.yaml.example` 复制到 `~/.ds-cli/ds.yaml.example`。
- 支持环境变量：`VERSION`、`PREFIX`、`SKILL_DIR`、`NO_SUDO`、`REPO`。

GitHub Actions：

- `.github/workflows/ci.yml`：push/PR 时执行 vet、gofmt、race test、build、help 和 skill front matter 检查。
- `.github/workflows/release.yml`：push 到 `main` 自动递增 patch tag，并在同一轮用 GoReleaser 生成 release；推送 `v*` tag 也会触发 release。
- `.goreleaser.yaml`：打包 `linux/darwin` × `amd64/arm64`，tar.gz 包含 README、LICENSE、`ds.yaml.example` 和 `skills/**/*`。

注意：如果只有 tag 没有 release artifact，一键安装会失败。安装脚本必须从 GitHub release 下载，不从源码构建。

## 代码结构

- `main.go`：入口，创建并执行 Cobra root command。
- `cmd/`：生命周期命令层，负责加载配置、创建 runlog、执行步骤并输出 JSON envelope。
- `internal/config`：加载 `ds.yaml`、合并默认值、校验配置。
- `internal/workflow`：生成本机部署需要的 bash 脚本，要求尽量幂等。
- `internal/local`：通过 `bash -lc` 执行步骤，记录每个步骤的 stdout/stderr。
- `internal/output`：定义 stdout JSON envelope。
- `internal/packages`：维护 DolphinScheduler、ZooKeeper、MySQL Driver 下载地址。
- `internal/render`：渲染 `dolphinscheduler_env.sh` 等配置内容。
- `internal/runlog`：写入 `~/.ds-cli/runs/<run-id>/`。
- `skills/dolphinscheduler-pseudo-cluster`：Claude Code 驱动 `ds-cli` 的 skill。

## 输出契约

- stdout 只能输出机器可读 JSON envelope，不要打印自由文本。
- stderr 用于人类可读进度。
- 每次运行写入 `~/.ds-cli/runs/<run-id>/`，失败时优先查看 `<step>.stderr`。
- 失败命令也应尽量输出完整 envelope，并返回非零退出码。

## 配置约定

配置查找顺序：

```text
--config <path> -> $DSCLI_CONFIG -> ./ds.yaml -> ~/.ds-cli/ds.yaml
```

`ds.yaml.example` 是用户起步模板。新增用户可见配置时，需要同步更新：

- `internal/config/types.go`
- `internal/config/load.go`
- `ds.yaml.example`
- `README.zh-CN.md`
- `skills/dolphinscheduler-pseudo-cluster/SKILL.md`

## 开发注意事项

- 保持本机部署边界，除非明确要求，不要加入 SSH、多节点 inventory 或 agent 逻辑。
- 生命周期命令应保持幂等：重复执行 `install/configure/start` 不应破坏已有部署。
- 不要把 `bin/`、`dist/` 等构建产物提交到仓库。
- 修改安装脚本后必须执行 `bash -n scripts/install.sh`，必要时用临时目录验证 `PREFIX` 和 `SKILL_DIR`。
