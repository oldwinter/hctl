# hctl

kubectl 风格的 AI 编程 agent **harness 控制面**（1.0.1）：在多台机器上 **读取、对比、安全改写、同步** Codex / Claude Code / Grok Build / Hermes / OpenCode / pi / droid / cursor-agent 的配置、默认模型和供应商。

主二进制是 **`hctl`**；`harnessctl` 仍作为同功能的次级入口。模块路径是 `github.com/oldwinter/hctl`，**不是** Rust 项目 `github.com/oldwinter/harnessctl`（声明式 harness 配置，互不覆盖）。

**不是调度器。** 不启动 agent，不替代 `all-cli`，不依赖 Herdr runtime。

## kubectl 对照

| kubectl | hctl |
| --- | --- |
| Context = 集群 | **Context = 环境 / 机器**（本机 `mba`、SSH `box`） |
| Resource | `harness` / `model` |
| get / describe / diff / apply / config | 同名语感命令 + `set` / `sync` / `doctor` |

## 安装

仓库：https://github.com/oldwinter/hctl （模块 `github.com/oldwinter/hctl`）。**不要**推到 / 安装自 Rust 项目 `oldwinter/harnessctl`。

```bash
# 主二进制
go install github.com/oldwinter/hctl/cmd/hctl@v1.0.1
# 或跟踪 main
go install github.com/oldwinter/hctl/cmd/hctl@latest

# 次级入口（同功能）
go install github.com/oldwinter/hctl/cmd/harnessctl@v1.0.1
```

`go install` 不会注入 git commit / build date；`hctl version` 会显示 `commit: unknown` / `built: unknown`。要带元数据，从源码用 just（见下）。

从源码编译：

```bash
git clone https://github.com/oldwinter/hctl.git
cd hctl
just build          # bin/hctl + bin/harnessctl，ldflags 注入 version/commit/date
just release        # dist/hctl_linux_amd64 + dist/harnessctl_linux_amd64 + SHA256SUMS
./bin/hctl version  # 例如：hctl version 1.0.1 / commit: abc1234 / built: 2026-...
```

发布产物（`just release`）：

| 文件 | 说明 |
| --- | --- |
| `dist/hctl_linux_amd64` | 主二进制（linux/amd64） |
| `dist/harnessctl_linux_amd64` | 次级入口（linux/amd64） |
| `dist/SHA256SUMS` | SHA-256 校验和 |

```bash
cd dist
sha256sum -c SHA256SUMS
```

Homebrew tap 不在 1.0 范围。

## 快速开始

```bash
hctl version
hctl config get-contexts
hctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses
hctl --no-probe --home testdata/home-a --config testdata/harnessctl.yaml --json get harnesses
hctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses -o wide
hctl --home testdata/home-a --config testdata/harnessctl.yaml --json get models
hctl --home testdata/home-a --config testdata/harnessctl.yaml -o json get models
hctl --home testdata/home-a --config testdata/harnessctl.yaml get harness codex
```

环境变量优先读 `HCTL_*`，再回退旧名 `HARNESSCTL_*`：`HOME`、`CONFIG`、`BACKUP_DIR`、`SSH=0`（测试时禁止真 SSH）。

配置文件只选一个，就地读写，**不会**复制或删除 `~/.harnessctl`：

1. `--config`
2. `$HCTL_CONFIG` / `$HARNESSCTL_CONFIG`
3. 已存在的 `~/.hctl/config.yaml`
4. 已存在的 `~/.harnessctl/config.yaml`
5. 否则新建 `~/.hctl/config.yaml`

备份目录：`$HCTL_BACKUP_DIR` / `$HARNESSCTL_BACKUP_DIR`，否则是解析后配置文件旁边的 `backups/`。

## 配置 mba / box

默认文件：`~/.hctl/config.yaml`（若只有旧文件则继续用 `~/.harnessctl/config.yaml`）。

```bash
hctl config set-context mba --kind local
hctl config set-context box --kind ssh --ssh you@box.example --home /home/you --identity ~/.ssh/id_ed25519
hctl config use-context mba
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

SSH 走本机 `ssh`：`BatchMode=yes`、`ConnectTimeout=8`，整段命令 10s 超时。远程读写用 `cat` / `mv`；`doctor` 用远程 `command -v` 判断是否安装（不远程跑 `--version`）。备份仍落在**本机**（见上面的备份目录规则）。

## 命令矩阵

| 命令 | 作用 | 退出码 |
| --- | --- | --- |
| `version` | 版本 / commit / date | 0 |
| `config get-contexts` / `current-context` / `use-context` / `set-context` | 环境 | 0 / 1 |
| `get harnesses` / `get models` / `get harness NAME` | 库存表；`--json` 与 `-o json` 等价；`-o wide` 加配置路径 | 0 |
| `describe harness NAME` | 单条快照 | 0 |
| `doctor` | 安装 / 配置 / 密钥 / onboarding / **key-drift**（人表有 DRIFT 列） | 0；解析错误为 5 |
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
hctl --home testdata/home-a set model codex o4-mini --dry-run
hctl --home testdata/home-a set model codex o4-mini

hctl --home testdata/home-a apply -f testdata/desired.toml --dry-run
hctl --home testdata/home-a apply -f testdata/desired.toml

hctl sync --from mba --to box --harness codex,claude --dry-run
hctl sync --from mba --to box --harness codex --fields model,provider,secret-ref
```

`--fields secret` 会把 bearer **字节**拷到对端（SSH 管道，不写日志），屏幕上只出现指纹。能用 `secret-ref`（环境变量名）就不要拷密钥。

`--no-probe` 保留本地 `PATH` / SSH `command -v` 安装检测和配置文件读取，但跳过本地 `--version` 与 `cursor-agent status` 子进程。它适合离线 inventory / observation；普通命令默认行为不变。

**`set provider` 诚实行为：** Claude 的供应商是隐式 anthropic，Grok 从 `base_url` 推断，Factory Droid 当前 custom models 不写 provider。对这些 harness 执行 `set provider`（含 `--dry-run`）会立刻返回用法错误，而不是静默成功后再在 verify 里失败。未知的 Codex `model_providers` / Hermes `providers` / Pi `models.json` 名同样立刻拒绝。Grok 改 default 时若目标没有 `[model."…"]` 段，会从当前 default 表复制后再切换，避免观察层丢 host/secret。

### dotfiles ownership

`set`、`apply`、`sync`（包括 `secret`）写入前会在**目标 context 的文件系统**中按以下顺序查 ownership：

1. `--ownership-manifest ABSOLUTE_OR_~/PATH`；
2. `~/.config/harness/ownership.json` 指向的 canonical manifest；
3. 仅当 pointer 不存在时，回退到 `~/dotfiles/harness/manifest.json`。

pointer 格式固定为：

```json
{"version":1,"owner":"oldwinter/dotfiles","manifest":"/absolute/path/to/manifest.json"}
```

manifest 使用 `version: 1` 和 `units[].dest`；每个 destination 必须是无 traversal 的 canonical `~/...` 路径。pointer、显式 manifest 或其内容只要存在但无效，就会 fail closed，`--dry-run` 与 `--allow-managed` 也不会绕过无效 ownership 配置。pointer 与 conventional manifest 都不存在时，目标按 standalone 处理。

只要某 adapter 的任一 `ConfigRelPaths` 被 manifest 管理，整个 adapter 的写入都会保守拒绝。`--allow-managed` 只临时绕过已经确认的 managed ownership。ownership guard 支持 local 与 SSH 目标；source 端只读，不做 ownership 检查。

`apply` / `sync` 会先完成整个目标 batch 的 ownership 检查与已实现的 adapter 预检，再开始 backup / write。开始执行后，每个文件仍按「唯一备份 → 临时文件 → rename → 再读校验」处理；后续 edit、I/O 或校验失败时不会自动回滚已经完成的更早写入。secret 写入也使用同一备份与再读校验流程。备份名包含 harness、源文件 basename、路径 hash、纳秒时间与随机后缀，避免同一批多文件或并发写入碰撞。

## 安全

- **永远不打印明文 API key / token。**
- 内联密钥只保留 `sha256` 前 8 位，表格为 `sha256:deadbeef`。
- host 去掉 userinfo / path。
- 渲染层再滤一层 `sk-…` / 长 hex。
- `testdata/` 只有 `sk-test-aaa` / `sk-test-bbb`。
- 同一 `BaseURLHost` 上指纹不一致时，`doctor` 按 host 分组并标 `key-drift`。

## Claude onboarding

`doctor` 会读 `~/.claude.json`（不仅是 `~/.claude/settings.json`）：

- 文件缺失 → `onboarding=needed`
- 缺少 `theme` 或 `hasCompletedOnboarding` → `onboarding=needed`
- `settings.json` 只剩 theme、没有 model → 额外提示 wizard leftover

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
| claude | `~/.claude/settings.json` + `~/.claude.json` |
| grok | `~/.grok/config.toml` |
| hermes | `~/.hermes/config.yaml` + `.env` |
| opencode | `~/.config/opencode/opencode.jsonc` |
| pi | `~/.pi/agent/settings.json` + `models.json` + `auth.json` |
| droid | `~/.factory/settings.json` |
| cursor-agent | `~/.cursor/cli-config.json`（本地 `cursor-agent status` 800ms 超时探测登录） |

`--json` 字段见 [`docs/json-schemas.md`](docs/json-schemas.md)。

Pi 读取当前 `settings.json` 的 `defaultProvider` / `defaultModel`，并从 `models.json.providers[defaultProvider]` 读取 endpoint。持久配置中的密钥观察优先级是所选 provider 的 `auth.json` credential（包括 credential 自带的 exact env mapping），其次是该 provider 的 `models.json.apiKey`；不会执行 command-based key 表达式，也不会把 OAuth credential 改写成 API key。Pi 运行时的 CLI 参数与 ambient environment precedence 不属于配置文件 observation。

Factory Droid 当前 `sessionDefaultSettings.model` / `customModels` schema 支持读取和 model 写入；provider、secret-ref 与 secret 写入会显式返回 unsupported。Hermes 当前 `custom_providers` 可读取；inline `api_key` 的改写/覆盖会显式拒绝。Hermes 在 provider 名不匹配时只使用唯一的完整 normalized endpoint 匹配，不按 host 猜测 credential。

## 开发

```bash
just test
just cover
just race
just lint
just smoke
just release
```

`just build` / `just release` 通过 `-ldflags` 写入 `internal/cli.Version` / `Commit` / `Date`（见 `justfile`）。裸 `go build` / `go install` 时 commit 与 date 为 `unknown`。

`just lint` 需要 **golangci-lint v2.13.2**（`.golangci.yml` 的 `version: "2"`）。安装见 [CONTRIBUTING.md](CONTRIBUTING.md)。

`just cover` 跑全量测试并检查 statement coverage 下限（当前 58.0%，见 `scripts/check-coverage.sh`）。GitHub Actions 对 push/PR 跑 gofmt、golangci-lint、race、同一覆盖率下限，并编译 `hctl` / `harnessctl`。

## 非目标

- 不调度 agent（herdr-orchestrator）
- 不替换 all-cli
- 1.0 不做 brew tap / GUI

## License

MIT
