# gosa Agents.md — go→sla→sa 新线（2026-11-25 立）

旧线（go2sa 直发 SA）封存在分支 legacy/go2sa-direct-sa，其 AGENTS.md 随分支保留、本文件取代之。

1. 发射目标只有 SLA 文本：前端产出 .sla（落盘可查），禁止直发 SA、禁止手改生成的 .sla。
2. 语义复用 SLA：所有权/检查/chan/select/动态值走 SLA 编译器与 sla_std，前端不重造 lowering 机器。
3. 前端实现用 Go（go/parser + go/types 是唯一正解）；CLI 保持薄，领域翻译按文件分工，单文件不堆巨无霸。
4. 禁静默错码：Go 语义在 SLA 无对应面时定位拒收（expect.refused 指纹），永不近似、永不假装支持。
5. demos/ 是验收规格：每步以 demos/run.sh（新管线版）差分门为准，步步独立 commit、零回退。
6. 分支纪律：legacy/go2sa-direct-sa 锁定不动；新线只在 main 推进。
