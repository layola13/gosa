# gosa Agents.md — go→sa 底座复用铁律（参考 tsgosa，2026-10-18 立项）

> 目标：在 `gosa`（新建仓，不克隆 Go；Go 源码浅克隆于 `/content/go` 只做移植源）内做 `go → sa`，
> 输出 `.sa` 给 `sci/sa` 消费，新增编译目标 `go2sa`。
> 知识底座：`sala`（只读知识库）、`sci`（Zig 核心编译器 + `sa_std` + `sa` CLI，后端消费者）、
> `tsgosa`（已有 `ts→sa` 链路/门禁/发射形状，移植源与范本）、`/content/go`（golang 浅克隆，`go/parser`+`go/types` 底座参考）。
> Go 源码仓只读：gosa 优先用标准库 `go/parser`+`go/types` 做前端，不整体搬迁 Go 源码。

## 1. 六条铁律（违反即回滚，与 tsgosa 同构）

1. **必须在已有架构做**：只改 `gosa` 已有文件（`internal/go2sa/*.go` lowering、`cmd/go2sa/main.go` 薄 CLI）。不另起平行 lowering 体系。
2. **必须复用底座**：SA 发射形状以 `tsgosa/demos/*.sai` 为范本（归属 `!t` 释放、`@import` 映射、`EXPAND` 宏）；`sci/sa_std` 现货符号直接映射，禁造轮子。
3. **禁止新建文件**：Go 源码零新文件（`demos/` fixture 例外：`main.go`/`expected.stdout`/`expect.refused`/`main.sa`/`run.sh`/README 可新建）。
4. **禁止原创逻辑**：语义以 Go 上游为准（`go/parser` AST → `go/types` 类型 → SA 发射）；无符号一律定位拒绝，永不静默错码、永不自造 helper/调用惯例。
5. **禁止在 MAIN.GO 里堆逻辑**：`cmd/go2sa/main.go` 只留 `runMain` 薄分发（flag 解析 + 调已有 internal 入口）；新增超 20 行即违规。
6. **禁止在 go2sa.go 写巨无霸**：`internal/go2sa/go2sa.go` 只留 `TranspileSA` 编排 + `RunSA` 薄 CLI；领域 lowering 按归属落 `lower_*.go`（后续由 JEV 落件）。

## 2. 管线（与 tsgosa 同构，前端换 Go）

```
.go --[parser+types: go/parser+go/types]--> 分析
    --[middle: desugar]--> 中转（原生 Go 形）
    --[emit: SA text + sci 宏]--> .sa + subset-report.txt
    --[sci sa build]--> .exe/.wasm
```

- 单文件口径（`TranspileSA`，`main` 包优先；多文件 program-link 后阶段）。
- 每步独立 commit + 零回退门禁，不批量切。

## 3. 门禁

- `demos/run.sh`：`main.go -> go2sa --sa -> main.sa -> sa build-exe -> diff expected.stdout`；`--check` 逐字节比对（`.sa` 禁止手改）。
- `expect.refused`：预期拒收工程（真 Go 本体形），build 必须拒收且指纹全中。
- 每小节任务完成必须提交推送到 `https://github.com/layola13/gosa`（`GIT_TOKEN`），然后找 JEV-NEXT 分配任务。

## 4. 工具纪律

- 工具链优先：Go（`/tmp/go/bin`，1.24.0）与 `sa` 后端（`sci/zig-out/bin/sa`），先验再动代码。
- 读/查/改优先 `sa_vm_run`，禁终端 `cat/grep/ls/head/tail` 看代码；`git/diagnostics` 走宿主。
- **禁静默错码**：转不过即定位拒绝，不得照抄错码——铁律 4 高于一切。
- 有疑问问 JEV（`jev_diagnose/jev_choose`），每步完问 `jev_next` 找下个任务；禁停下来总结、禁反问用户。
