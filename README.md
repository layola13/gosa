# gosa — go → sla → sa frontend

Go 前端新线（2026-11-25 定）：`.go` → `.sla` →（sla 编译器）→ SA → 原生二进制。

旧线 `go2sa`（.go 直发 .sa）已封存于分支 `legacy/go2sa-direct-sa`，不再维护；
本仓 main 为新线。旧线的 demos 语料（main.go + expected.stdout + expect.refused）
全部保留为新线的差分验收规格；旧线生成的 main.sa 已随封存移出 main（分支上仍在）。

管线：
```
.go --[go/parser + go/types]--> 分析 --[emit SLA text]--> .sla
    --[sla build-exe]--> SA --> 原生二进制
```

前端只做 AST→AST 翻译与 Go 语义映射；类型检查、所有权推断、chan/select、
动态值（any）等语义面复用 SLA 编译器与其运行时，不在前端重造。

门禁：demos/run.sh 待按新管线重写（main.go -> emit .sla -> sla build-exe -> diff expected.stdout）；
转不过的形状用 expect.refused 定位拒收，永不静默错码。
