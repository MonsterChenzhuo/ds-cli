# CLAUDE.md

This repository contains `ds-cli`, a local-only Go CLI for deploying Apache DolphinScheduler 3.4.1 in pseudo-cluster mode for Claude Code workflows.

## Commands

```bash
make build
make test
make fmt
make tidy
make all
```

## Architecture

- `cmd/` defines Cobra lifecycle commands.
- `internal/config` loads `ds.yaml` and applies defaults.
- `internal/workflow` builds idempotent bash scripts for local steps.
- `internal/local` runs scripts through `bash -lc` and writes per-step logs.
- `internal/output` emits one JSON envelope on stdout.
- `skills/dolphinscheduler-pseudo-cluster` is the Claude Code skill for driving the CLI.

## Contract

- stdout is machine JSON only.
- stderr is human progress only.
- logs live under `~/.ds-cli/runs/<run-id>/`.
- v1 is direct local deployment only. Do not add SSH inventory behavior unless the CLI scope changes.
