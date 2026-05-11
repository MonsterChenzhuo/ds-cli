# ds-cli

`ds-cli` deploys Apache DolphinScheduler 3.4.1 in local pseudo-cluster or distributed mode.

Task plugins are installed through DolphinScheduler's official flow: render `conf/plugins_config`, run `bash ./bin/install-plugins.sh 3.4.1`, and verify the configured jars under `plugins/task-plugins/`.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/MonsterChenzhuo/ds-cli/main/scripts/install.sh | bash
```

Overrides:

```bash
VERSION=v0.1.0 PREFIX=$HOME/.local/bin NO_SUDO=1 bash scripts/install.sh
```

For Chinese documentation, see [README.zh-CN.md](./README.zh-CN.md).
