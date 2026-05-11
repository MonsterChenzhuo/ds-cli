# CLAUDE.md

本仓库是 `ds-cli`：一个面向 Claude Code 使用的单二进制 Go CLI，用于部署 Apache DolphinScheduler 3.4.1，支持本机伪集群和多机分布式两种模式。

## 项目范围

- 伪集群模式：`api-server`、`master-server`、`worker-server`、`alert-server` 都在本机。
- 分布式模式：通过 `hosts`、`ssh`、`roles` 在多台 Linux/macOS 机器上部署 DolphinScheduler 服务。
- 注册中心默认由 `ds-cli` 安装 ZooKeeper；分布式模式允许用户通过 `zookeeper.external_connect_string` 复用外部 ZooKeeper。
- 元数据库使用用户提供的 MySQL；默认假设库和账号已存在，只有配置 `mysql.create_database: true` 时才用管理员账号创建数据库。
- 默认安装 task 插件 `shell` 和 `python`，jar 落在 `$DOLPHINSCHEDULER_HOME/plugins/task-plugins/`。
- CLI 会安装或复用 JDK 11、按需安装 ZooKeeper、下载 DolphinScheduler 二进制包、下载 MySQL JDBC Driver、渲染配置并执行库表初始化。

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
- 将 release 包内的 `skills/` 安装到 `~/.claude/skills/`，这是 Claude Code 当前自动发现个人 skills 的目录。
- 将 `ds.yaml.example` 复制到 `~/.ds-cli/ds.yaml.example`。
- 支持环境变量：`VERSION`、`PREFIX`、`SKILL_DIR`、`NO_SUDO`、`REPO`。
- 安装后 Claude Code 需要重新启动或重新加载会话，才能看到新安装的 skill。
- 安装 CLI 和运行 Claude Code 必须是同一个系统用户。root 环境下 Claude Code 会读取 `/root/.claude/skills/`；普通用户环境下读取 `$HOME/.claude/skills/`。

GitHub Actions：

- `.github/workflows/ci.yml`：push/PR 时执行 vet、gofmt、race test、build、help 和 skill front matter 检查。
- `.github/workflows/release.yml`：push 到 `main` 自动递增 patch tag，并在同一轮用 GoReleaser 生成 release；推送 `v*` tag 也会触发 release。
- `.goreleaser.yaml`：打包 `linux/darwin` × `amd64/arm64`，tar.gz 包含 README、LICENSE、`ds.yaml.example` 和 `skills/**/*`。

注意：如果只有 tag 没有 release artifact，一键安装会失败。安装脚本必须从 GitHub release 下载，不从源码构建。发布后需要确认 latest release 里至少包含：

```text
ds-cli_<version>_linux_amd64.tar.gz
ds-cli_<version>_checksums.txt
ds.distributed.yaml.example
skills/ds/SKILL.md
skills/dolphinscheduler-pseudo-cluster/SKILL.md
```

## 代码结构

- `main.go`：入口，创建并执行 Cobra root command。
- `cmd/`：生命周期命令层，负责加载配置、创建 runlog、执行步骤并输出 JSON envelope。
- `internal/config`：加载 `ds.yaml`、合并默认值、校验配置。
- `internal/workflow`：生成部署需要的 bash 脚本，要求尽量幂等；包含插件安装、逐服务 status、systemd unit 渲染。
- `internal/local`：通过 `bash -lc` 执行步骤，记录每个步骤的 stdout/stderr。
- `internal/remote`：SSH client、连接池和远程 runner；分布式模式使用它在 hosts 上并发执行任务。
- `internal/output`：定义 stdout JSON envelope。
- `internal/packages`：维护 DolphinScheduler、ZooKeeper、MySQL Driver 下载地址。
- `internal/render`：渲染 `dolphinscheduler_env.sh` 等配置内容。
- `internal/runlog`：写入 `~/.ds-cli/runs/<run-id>/`。
- `skills/dolphinscheduler-pseudo-cluster`：Claude Code 驱动 `ds-cli` 的完整 skill。
- `skills/ds`：`dolphinscheduler-pseudo-cluster` 的短别名，方便在 Claude Code 中通过 `/ds` 调用。

## 输出契约

- stdout 只能输出机器可读 JSON envelope，不要打印自由文本。
- stderr 用于人类可读进度。
- 每次运行写入 `~/.ds-cli/runs/<run-id>/`，失败时优先查看 `<step>.stderr`。
- 失败命令也应尽量输出完整 envelope，并返回非零退出码。
- `preflight` 失败必须把缺失工具或配置项写入 stderr，避免只出现 `exit status 1`。
- `status` 必须逐服务核对进程，不允许只要任意 DolphinScheduler 进程存在就判定成功。

## 配置约定

配置查找顺序：

```text
--config <path> -> $DSCLI_CONFIG -> ./ds.yaml -> ~/.ds-cli/ds.yaml
```

`ds.yaml.example` 是用户起步模板。新增用户可见配置时，需要同步更新：

- `internal/config/types.go`
- `internal/config/load.go`
- `ds.yaml.example`
- `ds.distributed.yaml.example`
- `README.zh-CN.md`
- `skills/dolphinscheduler-pseudo-cluster/SKILL.md`
- `skills/ds/SKILL.md`

## 插件与服务守护

- 默认 task 插件：`shell`、`python`。
- 插件下载地址来自 Maven Central，例如 `dolphinscheduler-task-python-3.4.1.jar` 和 `dolphinscheduler-task-shell-3.4.1.jar`。
- `install` 和 `bootstrap` 会安装默认插件。
- 用户需要补装或修复插件时，使用：

```bash
ds-cli plugins --restart
```

该命令会下载插件到 `plugins/task-plugins/`，并重启 `api-server`、`worker-server`。

服务静默退出的长期方案：

```bash
ds-cli systemd
```

该命令为声明的 DolphinScheduler 服务安装 systemd unit，并设置 `Restart=on-failure`。

## Skill 排障

如果安装后 Claude Code 中输入 `/ds` 或 `/dolphinscheduler-pseudo-cluster` 找不到 skill，按顺序检查：

```bash
which ds-cli
ds-cli --version
find ~/.claude/skills -maxdepth 2 -name SKILL.md -print
```

期望看到：

```text
~/.claude/skills/ds/SKILL.md
~/.claude/skills/dolphinscheduler-pseudo-cluster/SKILL.md
```

如果只看到 `~/.ds-cli/skills/...`，说明安装的是旧脚本或旧 release，需要重新执行最新安装脚本，或临时迁移：

```bash
mkdir -p ~/.claude/skills
cp -R ~/.ds-cli/skills/* ~/.claude/skills/
```

迁移后重启 Claude Code。

## 开发注意事项

- 分布式模式参考 `hadoop-cli` 的 inventory + SSH runner 思路，但 DolphinScheduler 角色使用 `zookeeper/api_server/master_server/worker_server/alert_server`。
- 生命周期命令应保持幂等：重复执行 `install/configure/start` 不应破坏已有部署。
- 不要把 `bin/`、`dist/` 等构建产物提交到仓库。
- 修改安装脚本后必须执行 `bash -n scripts/install.sh`，必要时用临时目录验证 `PREFIX` 和 `SKILL_DIR`。
