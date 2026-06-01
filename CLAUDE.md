# CLAUDE.md

本仓库是 `ds-cli`：一个面向 Claude Code/Codex 和自动化脚本的单二进制 Go CLI，用于通过 Apache DolphinScheduler REST API 操作已有 DS 集群。

## 项目范围

- 只做 REST API 操作，不负责安装、配置、启动、停止或升级 DolphinScheduler。
- 配置命名 API 集群 profile，默认写入 `~/.config/ds-cli/config.yaml`。
- 支持通过 profile、flag 和环境变量指定 API 地址与认证信息。
- API 操作覆盖项目、普通工作流、单任务工作流、调度、告警组和环境。
- API/profile 命令在 stdout 输出统一 JSON envelope，stderr 只用于错误诊断或 cobra 默认错误信息。

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
- 将 release 包内的 `skills/` 安装到 `~/.Codex/skills/` 和 `~/.claude/skills/`。
- 支持环境变量：`VERSION`、`PREFIX`、`SKILL_DIR`、`SKILL_DIRS`、`NO_SKILL`、`NO_SUDO`、`REPO`。

GitHub Actions：

- `.github/workflows/ci.yml`：push/PR 时执行 vet、gofmt、race test、build、help 和 skill front matter 检查。
- `.github/workflows/release.yml`：push 到 `main` 自动递增 patch tag，并在同一轮用 GoReleaser 生成 release；推送 `v*` tag 也会触发 release。
- `.goreleaser.yaml`：打包 `linux/darwin` × `amd64/arm64`，tar.gz 包含 README、LICENSE 和 `skills/**/*`。

注意：如果只有 tag 没有 release artifact，一键安装会失败。安装脚本必须从 GitHub release 下载，不从源码构建。发布后需要确认 latest release 里至少包含：

```text
ds-cli_<version>_linux_amd64.tar.gz
ds-cli_<version>_checksums.txt
skills/ds/SKILL.md
skills/dolphinscheduler-pseudo-cluster/SKILL.md
```

## 代码结构

- `main.go`：入口，创建并执行 Cobra root command。
- `cmd/`：命令层，包含 API profile 管理和各类 DolphinScheduler API 子命令。
- `internal/dsapi`：profile 配置、REST client、认证和单任务工作流 payload 生成。
- `internal/output`：定义 stdout JSON envelope。
- `skills/ds`：Codex/Claude Code 驱动 `ds-cli` 的短别名 skill。
- `skills/dolphinscheduler-pseudo-cluster`：保留旧 skill 名称用于兼容，但内容已收敛为 API 操作。

## 输出契约

- API/profile 命令的 stdout 只输出机器可读 JSON envelope，不要打印自由文本。
- stderr 不承载结果数据。
- API 命令失败时应尽量输出 envelope，并返回非零退出码。
- 配置错误使用 `CONFIG_ERROR`。
- DolphinScheduler API 请求错误使用 `DS_API_ERROR`。

成功示例：

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

## 配置约定

API profile 默认路径：

```text
~/.config/ds-cli/config.yaml
```

可通过 `DSCLI_CONFIG_DIR` 改变配置目录。配置结构：

```yaml
active_cluster: prod
clusters:
  prod:
    api_url: http://ds.example.com/dolphinscheduler
    username: admin
    password: dolphinscheduler123
    timeout: 30s
```

解析优先级：

- 集群：`--cluster` -> `DSCLI_CLUSTER` -> `active_cluster`
- API 地址：`--api-url` -> `DSCLI_API_URL` -> profile `api_url`
- 认证：`--token` / `DSCLI_TOKEN` -> `--session-id` / `DSCLI_SESSION_ID` -> 用户名密码
- 超时：`--api-timeout` -> `DSCLI_API_TIMEOUT` -> profile `timeout` -> `30s`

新增用户可见 API 配置或命令时，需要同步更新：

- `cmd/*_api.go`
- `internal/dsapi/*`
- `README.md`
- `README.zh-CN.md`
- `skills/ds/SKILL.md`
- `skills/dolphinscheduler-pseudo-cluster/SKILL.md`

## 开发注意事项

- 不要重新引入部署、SSH、ZooKeeper、MySQL 初始化、插件安装、systemd 或本地进程管理逻辑。
- API 命令保持非交互式；所有输入通过 flag、环境变量、profile 或文件。
- 对用户脚本内容使用文件读取或 JSON/form API，不要拼接 shell 命令。
- 修改安装脚本后必须执行 `bash -n scripts/install.sh`。
- 不要把 `bin/`、`dist/` 等构建产物提交到仓库。
