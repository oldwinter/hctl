# harnessctl

kubectl 风格的 AI 编程 agent **harness 控制面**（1.0）：在多台机器上 **读取、对比、安全改写、同步** Codex / Claude Code / Grok Build / Hermes / OpenCode / pi / droid / cursor-agent 的配置、默认模型和供应商。

**不是调度器。** 不启动 agent，不替代 `all-cli`，不依赖 Herdr runtime。

## kubectl 对照

| kubectl | harnessctl |
| --- | --- |
| Context = 集群 | **Context = 环境 / 机器**（本机 `mba`、SSH `box`） |
| Resource | `harness` / `model` |
| get / describe / diff / apply / config | 同名语感命令 + `set` / `sync` / `doctor` |

## 安装

模块路径：`github.com/oldwinter/harnessctl`。

若仓库还在 Origin 上，创建 GitHub 远程后再 `go install`：

```bash
go install github.com/oldwinter/harnessctl/cmd/harnessctl@v1.0.0
go install github.com/oldwinter/harnessctl/cmd/hctl@v1.0.0

# 或从源码
just build          # bin/harnessctl + bin/hctl
go build -o harnessctl ./cmd/harnessctl
go build -o hctl ./cmd/hctl
```

Homebrew tap 不在 1.0 范围；需要时再加 `brew tap oldwinter/tap`。

## 快速开始

```bash
harnessctl version
harnessctl config get-contexts
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses -o wide
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml --json get models
```

环境变量：`HARNESSCTL_HOME`、`HARNESSCTL_CONFIG`、`HARNESSCTL_BACKUP_DIR`、`HARNESSCTL_SSH=0`（测试时禁止真 SSH）。

## 配置 mba / box

默认文件：`~/.harnessctl/config.yaml`。

```bash
harnessctl config set-context mba --kind local
harnessctl config set-context box --kind ssh --ssh you@box.example --home /home/you --identity ~/.ssh/id_ed25519
harnessctl config use-context mba
```

```yaml
apiVersion: harnessctl/v1
kind: Config
current-context: mba
contexts:
  - name: mba
    context:
      kind: local
  - name: box
    context:
      kind: ssh
      ssh: you@box.example
      home: /home/you
      identityFile: /Users/you/.ssh/id_ed25519
```

SSH 走本机 `ssh`：`BatchMode=yes`、`ConnectTimeout=8`。远程读写用 `cat` / `mv`，备份仍落在**本机** `~/.harnessctl/backups/`。

## 命令矩阵

| 命令 | 作用 | 退出码 |
| --- | --- | --- |
| `version` | 版本 / commit / date | 0 |
| `config get-contexts` / `current-context` / `use-context` / `set-context` | 环境 | 0 / 1 |
| `get harnesses` / `get models` | 库存表；`--json` / `-o wide` | 0 |
| `describe harness NAME` | 单条快照 | 0 |
| `doctor` | 安装 / 配置 / 密钥 / onboarding / **key-drift** | 0；解析错误为 5 |
| `diff harness NAME --contexts mba,box` | 两边快照 | 0 / 4(ssh) |
| `diff harness NAME --home-a A --home-b B` | 两棵 home | 0 |
| `diff -f desired.toml` | 期望 vs 当前 | 0 |
| `set model\|provider NAME VALUE [--dry-run]` | 单字段写入 | 0 / 3(verify) / 2 |
| `apply -f FILE [--dry-run]` | 声明式写入 | 0 / 3 |
| `sync --from mba --to box --harness a,b [--fields …] [--dry-run]` | 跨 context | 0 / 4 |
| `completion bash\|zsh\|fish` | 补全 | 0 |

退出码：`0` 成功，`1` 通用，`2` 用法，`3` 写后校验失败，`4` SSH，`5` 解析错误。

## set / apply / sync

```bash
# 先看再写
harnessctl --home testdata/home-a set model codex o4-mini --dry-run
harnessctl --home testdata/home-a set model codex o4-mini

harnessctl --home testdata/home-a apply -f testdata/desired.toml --dry-run
harnessctl --home testdata/home-a apply -f testdata/desired.toml

# 跨环境（两边都配置好之后）
harnessctl sync --from mba --to box --harness codex,claude --dry-run
harnessctl sync --from mba --to box --harness codex --fields model,provider,secret-ref
```

`--fields secret` 会把 bearer **字节**拷到对端（SSH 管道，不写日志），屏幕上只出现指纹。能用 `secret-ref`（环境变量名）就不要拷密钥。

### dry-run sync 示例（已脱敏）

```
dry-run: no files written
HARNESS  FIELD  FROM           TO              PATH
codex    model  o4-mini        gpt-5.2-codex   -
secret codex action=bearer from=sha256:b6310a05
```

写入是「备份 → 临时文件 → rename → 再读校验」。备份：`~/.harnessctl/backups/<harness>-<timestamp>.bak`。

## 安全

- **永远不打印明文 API key / token。**
- 内联密钥只保留 `sha256` 前 8 位，表格为 `sha256:deadbeef`。
- host 去掉 userinfo / path。
- 渲染层再滤一层 `sk-…` / 长 hex。
- `testdata/` 只有 `sk-test-aaa` / `sk-test-bbb`。

## 格式保留（写入时）

| 格式 | 行为 |
| --- | --- |
| TOML | 按行改 key，**保留注释和无关表**；新 key 追加 |
| YAML | yaml.v3 node，尽量保留未改 key 的注释 |
| JSON | 2 空格重排，**键顺序可能变** |
| JSONC | **注释和尾逗号会丢**，写出标准 JSON |

## 适配器

| 名称 | 路径 |
| --- | --- |
| codex | `~/.codex/config.toml` |
| claude | `~/.claude/settings.json` |
| grok | `~/.grok/config.toml` |
| hermes | `~/.hermes/config.yaml` + `.env` |
| opencode | `~/.config/opencode/opencode.jsonc` |
| pi | `~/.pi/agent/settings.json` + `auth.json` |
| droid | `~/.factory/settings.json` |
| cursor-agent | `~/.cursor/cli-config.json`（`cursor-agent status` 短超时探测登录） |

`--json` 字段见 [`docs/json-schemas.md`](docs/json-schemas.md)。

## 开发

```bash
just test
just race
just lint
just smoke
```

## 非目标

- 不调度 agent（herdr-orchestrator）
- 不替换 all-cli
- 1.0 不做 brew tap / GUI

## License

MIT
