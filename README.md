# harnessctl

kubectl 风格的 AI 编程 agent **harness 控制面**：盘点、对比、诊断 Codex / Claude Code / Grok Build / Hermes / OpenCode（以及 pi / droid / cursor-agent 的占位）在多台机器上的 **配置、默认模型和供应商**。

v0.1 只做只读库存 + diff + doctor，可安装、可对 fixture / `--home` 跑通。

English summary: a kubectl-style, read-only inventory for coding-agent harness configs across environments. It does not dispatch agents and never prints raw API keys.

## 为什么单独成仓

这不是 `all-cli`，也不是 `herdr-orchestrator`。

| 项目 | 职责 |
| --- | --- |
| **harnessctl** | harness **配置** 的控制面（get / describe / diff / doctor） |
| herdr-orchestrator | **调度** agent 跑任务 |
| all-cli | 日常 CLI 工具箱，不要把 harness 配置逻辑混进去 |

## kubectl 对照

| kubectl | harnessctl |
| --- | --- |
| Context = kube 集群 | **Context = 环境 / 机器**（如本机 Mac `mba`、远程 Linux `box`） |
| Resource = pod / deploy | **Resource = `harness` / `model`**（后续 `provider`、`secret` 仅作引用） |
| `kubectl get` / `describe` / `diff` / `config` / `doctor` | 同名语感的子命令 |

v0.1 只在 **local** context 上读盘。`kind: ssh` 的 context 可以写进配置文件，执行时会给出明确的 “not implemented yet”（v0.2）。

## 安装

模块路径：`github.com/oldwinter/harnessctl`。

当前仓库若还在 Origin 临时项目上，请把 `go.mod` 的 module 视为目标路径；创建 GitHub / Origin 正式仓库后按该路径 `go install`。

```bash
# 源码构建（同时得到别名 hctl）
go build -o harnessctl ./cmd/harnessctl
go build -o hctl ./cmd/hctl

# 或
just build   # 产出 bin/harnessctl 与 bin/hctl

# 安装到 GOPATH/bin
go install github.com/oldwinter/harnessctl/cmd/harnessctl@latest
go install github.com/oldwinter/harnessctl/cmd/hctl@latest
```

不依赖 Herdr runtime，也不需要真实用户 harness 配置：用 `--home` 或 `HARNESSCTL_HOME` 指向任意 home 树即可。

```bash
go test ./...
go build -o harnessctl ./cmd/harnessctl
./harnessctl version
./harnessctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses
```

## 配置文件（类 kubeconfig）

默认路径：`~/.harnessctl/config.yaml`（可用 `--config` / `HARNESSCTL_CONFIG` 覆盖）。

文件不存在时，内置默认 **current-context = `mba`（local）**。`config use-context` 会把文件写出来。

```yaml
apiVersion: harnessctl/v1
kind: Config
current-context: mba
contexts:
  - name: mba
    context:
      kind: local
      # home: /Users/you          # 可选，覆盖扫描用的 $HOME
  # 如何加入远程 Linux box（v0.1 仅记录，不 SSH）：
  - name: box
    context:
      kind: ssh
      ssh: user@box.example
      home: /home/user
```

`kind: ssh` 在 v0.1 执行任何读盘命令都会报错，并提示用 `--home` 或 `--home-a` / `--home-b` 对比两棵目录。跨机器 diff 是 v0.2。

## 命令示例

```bash
harnessctl version
harnessctl config get-contexts
harnessctl config current-context
harnessctl config use-context mba

# 指向 fixture，无需本机真实 ~/.codex 等
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml --json get harnesses
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml get models
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml describe harness codex
harnessctl --home testdata/home-a --config testdata/harnessctl.yaml doctor

# 对比两个 home / fixture（跨机器 SSH diff = v0.2）
harnessctl diff harness codex --home-a testdata/home-a --home-b testdata/home-b
harnessctl diff harness codex --a mba --b box --home-a testdata/home-a --home-b testdata/home-b
# 两边都是 local 时也可以：
# harnessctl diff harness codex --contexts mba,other-local
```

环境变量：`HARNESSCTL_HOME`、`HARNESSCTL_CONFIG`。报告类命令支持 `--json`。

## 读到的 harness 格式（v0.1）

| 名称 | 配置 | 抽取字段 |
| --- | --- | --- |
| `codex` | `~/.codex/config.toml` | `model`, `model_provider`, `[model_providers.*].base_url`, bearer / `env_key` |
| `claude` | `~/.claude/settings.json` | `model`, `env.ANTHROPIC_*`（含角色 alias） |
| `grok` | `~/.grok/config.toml` | `[models].default`, `[model."…"].base_url` / `api_key` |
| `hermes` | `~/.hermes/config.yaml` + `.env` | `model.default` / `providers.*.base_url` / `key_env` |
| `opencode` | `~/.config/opencode/opencode.jsonc` | `model`（`provider/model`）、`provider.*.options` 或 `providers.*.settings` |
| `pi` / `droid` / `cursor-agent` | 常见路径探测 | 仅占位，完整 parser 后续再加 |

统一快照：名称、可执行文件路径/版本（PATH 上能检测到才有）、配置路径、供应商、**base URL 的 host**（不要完整密钥）、默认模型、effort / alias、密钥指纹。

## 安全

- **永远不打印明文 API key / token。**
- 若配置里有内联密钥，只保留 `sha256(key)` 的前 8 位十六进制，表格里显示为 `sha256:deadbeef`。
- 若只写了环境变量名或 `{env:NAME}`，显示 `env:NAME`。
- host 字段去掉 userinfo 和 path，避免 `https://user:token@host/v1` 泄漏。
- 渲染层还有一层 `sk-…` / 长 hex 的 redact 兜底。
- `testdata/` 只用假密钥：`sk-test-aaa` / `sk-test-bbb`。

## 开发

```bash
just test
just fmt
just lint    # 有 golangci-lint 则用；否则 go vet
just smoke
just build
```

包布局：`cmd/`、`internal/cli`、`internal/config`、`internal/model`、`internal/adapters/<harness>`、`internal/render`、`testdata/`。

## 路线图 v0.2

- `set` / `apply` / `sync`：写回各 harness 的真实配置（仍不打印密钥）。
- SSH context：对 `box` 做远程读（以及真正的跨机器 `diff`）。
- `provider` / `secret` 资源（secret 仅引用与指纹）。
- pi / droid / cursor-agent 的完整只读适配器。

## License

MIT
