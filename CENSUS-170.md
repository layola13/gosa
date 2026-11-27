# gosla Go 语义面普查 + 增量计划（集成地图）

任务 170 第一交付 · 修订版（R1–R6，审阅 APPROVE WITH NOTES 后） · 2026-11-26

> 修订说明（2026-11-26）：按小火车审阅 REVIEW-170-census.md（APPROVE WITH
> NOTES）修订：R1–R4 必修、R5/R6 建议均已处理；5 个问题逐一答复见文末
> 「审阅答复」一节；§10 重数已重算（语言面 139 → 142 项）。

> 口径：对照 Go 语言参考的语义面逐项核 gosla 现状。现状以 `internal/gosla`
> 源码为准（`gosla.go` 的 `slaType`/`funcDecl`/`genDecl` +
> `emit_expr.go` / `emit_stmt.go` / `emit_chan.go`），不凭印象；
> 每项的验收 demo 见「验收」列，demos 门以 `demos/run.sh` 为准。
> 格式照 sapy `SYNTAX-CENSUS.md` 的做法（逐项 支持/部分/缺口 + 汇总重数）。
>
> 状态：
> - **支持** = 前端已实现且经 demo 端到端真跑过（`demos/run.sh` 绿）。
> - **部分** = 有限形态可用，其余形态响亮拒绝或未验证。
> - **缺口-前端** = 纯前端可做，不依赖 SLA 本体改动（写手线/集成阶段的工作）。
> - **缺口-SLA** = 须上游 SLA 源级表面，gosla 不得自行绕过（响亮拒绝，登记上报）。
> - **排除** = v1 明确不做、响亮报错。
>
> 基线：`layola13/gosa` 远端头 `c9fe2bd4`；盒上工作副本
> `/content/sunny-work/gosa`，分支 `task-170-census`，HEAD `a4503d5`
> （普查提交本身，未推远端），其父提交即远端头 `c9fe2bd4`；
> 增量 1–3 已在头里（goroutine/chan/select，28/28 demos 绿）。
> （R3：原稿“HEAD 与远端一致”表述不准，已更正；原提交信息
> “(148 items)”与文件实数不符，修订版为 142 项——见 §10。）
> 本地提交 `596ac27` 不在远端也不在盒上（gosa/satsgo/tsgosa 均查无此提交），
> 应是主会话侧待推的本地提交；本普查以远端头 `c9fe2bd4` 为准，
> 如 `596ac27` 含新增语义面，推上来后补普查。
>
> 分工（muse DM 13185 / DM 13216，用户亲裁）：
> 小火车任线主；Sunny 负责普查 + gostd 集成 + demo 核验。
> 前端沿 Go 本体继续（`go/parser` + `go/types`），裁定标准 = 速度；
> 写手线的 Rust 稿不作废，按 Go 模块布局 1:1 移植作设计稿，
> 规则表与拒绝文案直接搬。本普查与集成地图按 Go 模块布局对齐（不按 Rust 布局）。
> 修订版交小火车复核重数，确认后首报 muse。
>
> std 扩面与移交（DM 13239 / DM 13242，用户 15:53/15:54 亲令）：
> §7 已扩到 Go std 全包面逐包状态（180 包）；主会话退出 gosla，
> Sunny 与小火车合作完成全程，staging 已移交（见 §11）。
>
> 门禁复核（本次普查实测，2026-11-26 ~15:40 CST，盒 `dfc9f8f7930a`）：
> `sa 0.1.11-优化`（`/content/sa_all/sci/zig-out/bin/sa`）+
> Go 1.27.2 + 隔离 HOME（sla 插件 0.1.8）+ `SA_STD_DIR=/content/sa_all/sci/sa_std`
> + `GOSLA_SA`，`./demos/run.sh` = **28/28（19 PASS + 9 REFUSED-OK），GATE_RC=0**。
> 注意：`run.sh` 本身不设 `SA_STD_DIR`；不设时 19 个 build-exe 全败
> （`Import Error: failed to expand @import SLA sources: error.FileNotFound`），
> 属门禁环境缺口、非发射器缺陷——集成阶段须把 `SA_STD_DIR` 写进门禁调用（见 §11）。

## 1. 程序结构与声明（10 项：支持 4 / 部分 2 / 缺口-前端 4）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| package main 单文件 | 支持 | `Transpile` 只收 `package main`，他包拒收 | 01_hello |
| import 声明 | 部分 | `genDecl` 遇 IMPORT 直接放过、不记录；只有 `fmt.Println` 真可用，其余包的符号在调用点才拒收；未使用的他包 import 会静默漏过（响亮度缺口） | 01_hello |
| 顶层 func 声明 | 支持 | 普通函数；见 §5 | 08_funcs |
| 顶层 const 声明 | 支持 | 用点折叠（int/bool/string）；float const 拒收 | 17_const、20_floatrefuse_const（拒收） |
| 顶层 var 声明 | 缺口-前端 | `genDecl` 默认分支拒收 `top-level var declarations not yet supported`；局部 var 已支持 | 待补 demo（拒收） |
| 顶层 type 声明（struct） | 部分 | 仅 struct 类型声明；非 struct 的 type 声明拒收 | 11_struct |
| 方法（receiver） | 缺口-前端 | `funcDecl` 遇 `Recv!= nil` 拒收 `methods not yet supported` | 待补 demo（拒收） |
| init 函数 | 缺口-前端 | 未特判：会原样编成 `fn init()` 但永不被调用（静默死代码，违 AGENTS.md 规则 4）；须拒收或由 main 接线后才能放行 | 待补 demo（拒收；集成前置阻塞项，见 §11） |
| main 形态（无参无结果） | 支持 | main 须无参无结果，发射 `-> i32` + `return 0;` | 01_hello |
| 多文件包 | 缺口-前端 | `Transpile` 只收单个文件路径；跨文件符号、包内多文件均未做 | 待补 demo（拒收） |

## 2. 类型（17 项：支持 8 / 缺口-前端 9）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| int → i64 | 支持 | `slaType` 直映 | 02_arith |
| bool | 支持 | | 10_bool |
| string → Str | 支持 | 绑定/赋值/len/Println 通；拼接与下标见 §9（SLA 面） | 16_string |
| float32/float64 | 缺口-前端 | 字面量、签名、const 三路均拒收 | 18/19/20（拒收） |
| 其余整数类型（int8/16/32/64、uint*、byte、rune） | 缺口-前端 | 仅 `types.Int` 映射，其余在 `slaType` 落空拒收 | 待补 demo（拒收） |
| struct（具名；字段限 int/bool/string/slice/map/chan/array/struct） | 支持 | 字段名首字母小写化；字面量 keyed + positional | 11_struct |
| 匿名 struct 类型 | 缺口-前端 | `slaType` 的 Struct 分支要求 `*types.Named`，匿名 struct 拒收 | 待补 demo（拒收） |
| 嵌入字段 | 缺口-前端 | 未处理 | 待补 demo（拒收） |
| slice []T → Vec<T> | 支持 | make/index/append/len/range 全通 | 12_slice、13_append、14_range |
| array [N]T | 支持 | 字面量 keyed + positional、len、index | 21_array |
| map[K]V → HashMap | 支持 | make/insert/get（缺失键得零值）/delete/len | 15_map |
| chan T → Chan<T> | 支持 | 无缓冲 + 有缓冲（cap → `Chan::new(cap)`）；定向 chan 拒收 | 27_goroutine_chan |
| 指针 *T | 缺口-前端 | 不在 `slaType`；`&`、`*` 单目亦拒收 | 待补 demo（拒收） |
| interface / any | 缺口-前端 | 无映射；any 作结果类型拒收 | 28_any（拒收） |
| 函数类型 / 函数值 | 缺口-前端 | 无函数类型映射 | 待补 demo（拒收） |
| 具名类型别名（非 struct 的 type 定义） | 缺口-前端 | `genDecl` 拒收非 struct 的 type 声明 | 待补 demo（拒收） |
| 泛型（类型参数/约束/实例化） | 缺口-前端 | 无 | 待补 demo（拒收） |

## 3. 表达式（36 项：支持 21 / 部分 2 / 缺口-前端 11 / 缺口-SLA 2）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| int/string 字面量 | 支持 | | 02_arith、16_string |
| float 字面量 | 缺口-前端 | 拒收 `unsupported literal FLOAT` | 18_floatrefuse_floatlit（拒收） |
| rune 字面量 | 缺口-前端 | `token.CHAR` 不在 INT/STRING 之列，拒收 | 待补 demo（拒收） |
| bool 字面量 | 支持 | | 10_bool |
| nil | 部分 | 直通发射为 `nil`，未经 demo 端到端验证；SLA 的 nil 语义未核，须补 demo 或拒收 | 待补 demo |
| 标识符 + const 折叠 | 支持 | int/bool/string const 在用点折叠；其余 const 拒收 | 17_const |
| 一元 - / ! | 支持 | | 02_arith、10_bool |
| 一元 & * ^（取址/解引用/异或单目） | 缺口-前端 | 仅 SUB/NOT，其余拒收 `unsupported unary operator` | 待补 demo（拒收） |
| 二元算术 + - * / % | 支持 | int 运算；string 的 + 见下（SLA 面） | 02_arith |
| 二元比较 == != < <= > >= | 支持 | | 04_ifelse |
| 逻辑 && \|\| | 支持 | | 10_bool |
| 位运算 & \| ^ &^ << >> | 缺口-前端 | 拒收 `unsupported binary operator` | 待补 demo（拒收） |
| 括号 | 支持 | | 02_arith |
| slice/array 下标读/写 | 支持 | | 12_slice、21_array |
| map 下标读/写 | 支持 | 读缺失键得零值（`.get().copied().unwrap_or_default()`）；写走 `insert` | 15_map |
| string 下标 | 缺口-SLA | 拒收 `string indexing not yet supported`；须 SLA 源级 Str 字节读取面（SLA 本体事，gosla 不得绕） | 23_strindex（拒收） |
| string 拼接 | 缺口-SLA | 拒收 `string concatenation not yet supported`；须 SLA 源级 Str concat 面（SLA 本体事，gosla 不得绕） | 22_concat（拒收） |
| struct 字段选择 | 支持 | 首字母小写化 | 11_struct |
| 包选择子 | 部分 | 仅 `fmt.Println` 真可用；其余包限定的值（如 `pkg.Var`）未守卫，会漏到 SLA 层报错 | 01_hello |
| struct 复合字面量（具名，keyed + positional） | 支持 | 缺失字段补零值 | 11_struct |
| array 复合字面量（keyed + positional） | 支持 | key 须为常量下标 | 21_array |
| slice 复合字面量 | 缺口-前端 | `compositeLit` 只处理 Struct/Array，slice 字面量拒收 | 待补 demo（拒收） |
| map 复合字面量 | 缺口-前端 | 同上，拒收 | 待补 demo（拒收） |
| 普通函数调用 | 支持 | | 08_funcs |
| fmt.Println（单参） | 支持 | string 走 `println(s)`，其余走 `println("{}", v)` | 01_hello |
| fmt.Print/Printf/Sprintf、Println 多参 | 缺口-前端 | `Println` 非单参拒收；其余 fmt 函数无分发（gostd 文本半，见 §7） | 待补 demo（拒收） |
| len | 支持 | | 12_slice、15_map、16_string |
| make/append/delete（语句位） | 支持 | 表达式位调用拒收 `only supported in statement position` | 12_slice、13_append、15_map |
| 类型转换 T(x) | 缺口-前端 | `CallExpr` 的 Fun 非 Ident/Selector 即拒收 `unsupported call target` | 待补 demo（拒收） |
| 类型断言 x.(T) / 类型 switch | 缺口-前端 | 无处理分支 | 28_any（拒收，卡在结果类型） |
| 接收 <-ch（赋值位/select 位） | 支持 | 赋值位见 §4，select 位见 §6 | 27_goroutine_chan |
| 闭包 / 函数字面量 | 缺口-前端 | 无 `FuncLit` 分支，拒收 `unsupported expression` | 待补 demo（拒收） |
| 方法调用 x.M() | 缺口-前端 | 选择子调用除 `fmt.Println` 外一律拒收 `unsupported selector call` | 待补 demo（拒收） |
| 未定义标识符（go/types 前置拒收） | 支持 | Go 本身编译期错误；`go/types` 层定位拒收 `undefined: ...`，gosla 尚未接手即拒收，与 23_strindex（gosla 自身拒收，SLA 面）不是同一层；门禁 REFUSED-OK | 09_typerefuse（拒收） |
| const 串编译期下标越界（go/types 前置拒收） | 支持 | Go 本身编译期错误；`go/types` 层拒收 `index 5 out of bounds`，与 23_strindex（变量串下标读，gosla 拒收）不是同一层；门禁 REFUSED-OK | 24_strindex_oob（拒收） |
| 串索引写（go/types 前置拒收） | 支持 | Go 本身编译期错误（串不可变）；`go/types` 层拒收 `cannot assign to s[0]`；与 23_strindex（下标读）不是同一层；门禁 REFUSED-OK | 25_strindex_store（拒收） |

> R1 注（go/types 前置拒收）：以上 3 项是 Go 本身的编译期错误，正确行为
> 就是定位拒收；`go/types` 层（gosla 前端的一部分）已实现且 demo 端到端
> 跑绿（REFUSED-OK），无后续工作，故状态记**支持**（见文末 Q1 答复）。
> 它们与 23_strindex（gosla 自身拒收，须 SLA 面）不是同一层，不得互相覆盖。

## 4. 语句（31 项：支持 19 / 部分 1 / 缺口-前端 11）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| 局部 var 声明（有值/无值） | 支持 | 无值时零值（int/bool/string/array；slice/map/chan 须走 make，见 `zeroValue` 落空拒收） | 03_vars、26_fib |
| := / = 单赋值 | 支持 | | 03_vars |
| 多重赋值 | 缺口-前端 | 拒收 `multiple assignment not yet supported` | 待补 demo（拒收） |
| 复合赋值 += -= *= /= %=（变量/字段） | 支持 | | 03_vars、11_struct |
| 复合赋值到下标 | 缺口-前端 | 拒收 `compound assignment to an index not yet supported` | 待补 demo（拒收） |
| ++ / -- | 支持 | | 05_for、06_while |
| if / else if / else | 支持 | | 04_ifelse |
| if/switch 带 init 语句 | 缺口-前端 | 拒收 `if with init statement not yet supported` / `switch with init statement not yet supported` | 待补 demo（拒收） |
| for 三段式 | 支持 | 发射为 while + 尾部 post | 05_for |
| for 条件式（while 形） | 支持 | | 06_while |
| for {} 无限循环 | 部分 | 条件缺省为 `true` 已实现，但无限形无端到端 demo（06_while 只跑条件形）；按“支持”定义（须 demo 真跑过）改记部分，待补 demo 后转正 | 06_while（条件形；无限形待补 demo） |
| range over slice（key/value/_/break/continue） | 支持 | 计数器循环 + `continue` 时补增量 | 14_range |
| range over map | 缺口-前端 | 未按 go/types 守卫：当前会漏过去编出按整数键 0..len-1 迭代的错误代码（键不连续即错）；须在 gosla 层拒收（集成前置阻塞项，见 §11） | 待补 demo（拒收；先加守卫再补） |
| range over string/chan | 缺口-前端 | string：下标位已拒收（响亮）；chan：`len(ch)` + `ch[i]` 未守卫，漏到 SLA 层报错；须在 gosla 层拒收 | 待补 demo（拒收） |
| switch（tag、多 case、default、case 内 break/continue） | 支持 | 无 default 时补空 default；case 内 break 截断后续语句（Go 语义） | 07_switch |
| tagless switch | 缺口-前端 | 拒收 `tagless switch not yet supported` | 待补 demo（拒收） |
| type switch | 缺口-前端 | 无 `TypeSwitchStmt` 分支，拒收 `unsupported statement` | 待补 demo（拒收） |
| select（recv/send arms，带 ok） | 支持 | recv 臂 `case let (v, closed) = ch.recv()`，ok = `closed == false` | 27_goroutine_chan |
| select default 子句 | 缺口-前端 | 拒收 `select default clause not yet supported`（非阻塞语义尚无已验证的 SLA select 形） | 待补 demo（拒收） |
| go 语句 | 支持 | 仅普通函数 Ident 调用；循环体内拒收；块末 join（见 §6） | 27_goroutine_chan |
| ch <- v 发送语句 | 支持 | | 27_goroutine_chan |
| v := <-ch / v, ok := <-ch | 支持 | 仅 `:=` 形；向已存在变量赋值拒收 | 27_goroutine_chan |
| return（0/1 值） | 支持 | 多返回值拒收；带 live goroutine/chan 时拒收 | 08_funcs、26_fib |
| break/continue（无标签） | 支持 | range 内 `continue` 补增量 | 06_while、07_switch、14_range |
| break/continue（带标签） | 缺口-前端 | 拒收 `labelled break/continue not yet supported` | 待补 demo（拒收） |
| goto / fallthrough | 缺口-前端 | 无分支，拒收 `unsupported statement` | 待补 demo（拒收） |
| defer | 缺口-前端 | 无分支，拒收 `unsupported statement`（Go 语义核心，见 §10 价值序） | 待补 demo（拒收） |
| 块语句 | 支持 | 作用域进出带 chan/handle 清理 | 27_goroutine_chan |
| delete(m, k) | 支持 | | 15_map |
| 空标识符 _（接收目标/range 变量/赋值） | 支持 | | 14_range、27_goroutine_chan |
| 局部 const 声明（用点折叠） | 支持 | `DeclStmt` 遇 CONST 直接跳过（用点折叠） | 17_const |

## 5. 函数（10 项：支持 4 / 缺口-前端 6）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| 普通函数（单结果/无结果） | 支持 | | 08_funcs |
| 多返回值 | 缺口-前端 | 声明与 return 两处均拒收 `multiple results/results not yet supported` | 待补 demo（拒收） |
| void 函数 | 支持 | | 08_funcs（printTwice） |
| 参数（int/bool/string/slice/map/chan/array/struct） | 支持 | | 08_funcs、27_goroutine_chan |
| float/any/pointer/interface 参数 | 缺口-前端 | `slaType` 落空即拒收 `unsupported param type` | 19_floatrefuse_signature（拒收，float） |
| 递归 | 支持 | | 26_fib |
| 具名结果参数 | 缺口-前端 | 结果名未声明为局部量：体内赋值与空 `return` 会漏到 SLA 层报错；须 gosla 层拒收或接线（集成前置阻塞项，见 §11） | 待补 demo（拒收） |
| 可变参数 ...T | 缺口-前端 | `slaType` 会把 `...T` 当 slice 映射（静默错形）；须拒收 | 待补 demo（拒收） |
| 闭包/函数值作参数或返回值 | 缺口-前端 | 无函数类型映射 | 待补 demo（拒收） |
| 函数值调用（变量作函数） | 缺口-前端 | `call` 只认 Ident/Selector；变量调用会漏到 SLA 层报错；须 gosla 层拒收 | 待补 demo（拒收） |

## 6. 并发（17 项：支持 6 / 部分 1 / 缺口-前端 10）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| go f(args) | 支持 | 仅普通函数 Ident 调用；块末 join；循环体内拒收（见下） | 27_goroutine_chan |
| make(chan T) / make(chan T, cap) | 支持 | cap → `Chan::new(cap)`；块末 destroy | 27_goroutine_chan |
| 定向 chan（chan<- / <-chan） | 缺口-前端 | 拒收 `directed channel types not yet supported` | 待补 demo（拒收） |
| ch <- v | 支持 | | 27_goroutine_chan |
| v := <-ch / v, ok := <-ch | 支持 | ok = `closed == false` | 27_goroutine_chan |
| select recv/send arms | 支持 | 含 `_, ok` 形与裸接收臂 | 27_goroutine_chan |
| select default 子句 | 缺口-前端 | 拒收（非阻塞语义尚无已验证的 SLA select 形） | 待补 demo（拒收） |
| close(ch) | 缺口-前端 | 无处理：会漏成未知 SLA 调用 `close(ch)`，SLA 层才报错；须 gosla 层拒收（集成前置阻塞项，见 §11） | 待补 demo（拒收） |
| range over chan | 缺口-前端 | 未守卫，漏到 SLA 层报错；须 gosla 层拒收 | 待补 demo（拒收） |
| go 在循环体内 | 缺口-前端 | 拒收 `go statement inside a loop body not yet supported`（break 可能跳过块末清理） | 待补 demo（拒收） |
| make(chan) 在循环体内 | 缺口-前端 | 拒收 `make(chan) inside a loop body not yet supported` | 待补 demo（拒收） |
| return 时带 live goroutine/chan | 缺口-前端 | 拒收 `return with live goroutines/channels not yet supported` | 待补 demo（拒收） |
| 接收赋值到已存在变量 | 缺口-前端 | 拒收 `receive assignment to existing variables not yet supported` | 待补 demo（拒收） |
| select 内接收赋值（非 :=） | 缺口-前端 | 拒收 `select receive assignment (not :=) not yet supported` | 待补 demo（拒收） |
| sync 包（Mutex/WaitGroup/Once） | 缺口-前端 | 标准库，gostd 运行时半（见 §7） | 待补 demo（拒收） |
| 缓冲 chan 语义 | 部分 | cap 直通 SLA `Chan::new` 已实现，但有缓冲形无端到端 demo（27 只跑 cap=0）；按“支持”定义改记部分，待补 demo 后转正 | 27_goroutine_chan（cap=0；有缓冲形待补 demo） |
| 通道关闭检测（ok） | 支持 | 经 recv 的 closed 位取反 | 27_goroutine_chan |

## 7. 包与导入 / 标准库（183 项：std 包 180 + 包模型 3）

> 口径（DM 13239 扩面）：Go 1.27.2 `go list std` 全量 383 包，滤掉所有含
> internal 路径段的包（`internal/*`、`*/internal/*` 与 `*/internal`——
> 后者 5 个：database/sql/internal、encoding/json/internal、log/internal、
> log/slog/internal、net/http/internal；用户代码不可 import）与 vendor
> （stdlib 实现细节）后得可 import 的公开包 180 个，逐包标状态。
> （R4：原稿只写了前两种模式，漏了 `*/internal` 这一档，文字已更正；180 这个数不变。）
> internal/vendor 不计入缺口。
>
> 状态（std 轨专用，与 §1–§6/§8/§9 的前端状态不同）：
> - **已写** = gostd SLA 侧函数已写好（staging，主会话本机），勿重写；
> ≠可用——前端映射层 + demo 接线尚未做，未集成前 gosla 仍拒收其调用。
> - **待写** = 纯 Go / SLA 可表达，待 std 轨写；「第一波」为今晚目标的建议波次（待线主确认）。
> - **基底缺口-响亮拒绝** = 须 SLA 尚无的基底能力（OS/系统调用/网络、运行时反射、
> cgo、工具链自举、动态加载等）；gosla 遇上必须响亮拒绝，不得降级冒充。
>
> 今晚 std 轨目标（DM 13239）：普查完成 + 第一波核心包写完。
> 每个集成进来的包配一个 demo（见 §11.3）；映射名统一 `go_<pkg>_<Name>`，
> 包路径中的 `/` 在映射名里写作 `_`（如 `go_encoding_json_Marshal`；
> 顶层包直接 `go_<pkg>_<Name>`，如 `go_strings_Contains`）。

### 7.1 已写（8 包，gostd 两半）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| errors | 已写 | gostd 文本半；error 值惯例须接口/any 先行（§8） | go_errors_New / Is / As |
| fmt | 已写 | gostd 文本半；前端现状仅 Println 单参直通，其余须映射层接线 | go_fmt_Print / Printf / Sprintf / Sprintln / Errorf |
| math | 已写 | gostd 运行时半；全系 float，映射须 float 落地后才可全通 | go_math_Abs / Sqrt / Pow / Max / Min |
| sort | 已写 | gostd 运行时半 | go_sort_Ints / Strings / Slice / Sort |
| strconv | 已写 | gostd 文本半；float 相关须 float 落地后 | go_strconv_Itoa / Atoi / FormatInt / ParseInt / Quote |
| strings | 已写 | gostd 文本半；受 Str 面限制（§9，拼接/索引须 SLA 面先行） | go_strings_Contains / Index / Split / Join / 等 |
| sync | 已写 | gostd 运行时半；Mutex/WaitGroup/Once | go_sync_Mutex / WaitGroup / Once |
| time | 已写 | gostd 运行时半；Now/Sleep/Since/Duration | go_time_Now / Sleep / Since / Parse |

### 7.2 待写（103 包）

#### 7.2.1 第一波（建议，19 包——今晚目标；R6 拆 1a/1b）

> 波内顺序：io 先于 bufio/log；encoding 先于 encoding/json。
> 1b 的 6 包前置不满足就顺延到第二波，不硬塞进今晚。

##### 1a 无前置（13 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| bytes | 待写（第一波 1a） | 核心；Buffer/Compare/Equal/Fields/Join/Repeat/Trim | go_bytes_* |
| bufio | 待写（第一波 1a） | 须 io；Reader/Writer/Scanner | go_bufio_* |
| io | 待写（第一波 1a） | 核心接口（Reader/Writer/Seeker）+ Copy/ReadAll 等 helper | go_io_* |
| context | 待写（第一波 1a） | 须 goroutine/chan 面（已通，§6） | go_context_* |
| encoding/base64 | 待写（第一波 1a） | 纯编码 | go_encoding_base64_* |
| encoding/hex | 待写（第一波 1a） | 纯编码 | go_encoding_hex_* |
| hash | 待写（第一波 1a） | Hash 接口 | go_hash_* |
| hash/crc32 | 待写（第一波 1a） | 纯校验 | go_hash_crc32_* |
| hash/fnv | 待写（第一波 1a） | 纯校验 | go_hash_fnv_* |
| flag | 待写（第一波 1a） | 命令行解析，纯 | go_flag_* |
| log | 待写（第一波 1a） | 须 io；Print/Fatal/Panic（Fatal/Panic 语义待定） | go_log_* |
| path | 待写（第一波 1a） | 纯路径操作 | go_path_* |
| unicode/utf8 | 待写（第一波 1a） | 纯；DecodeRune/EncodeRune/RuneLen | go_unicode_utf8_* |

##### 1b 有前置（6 包；前置逐个写明）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| encoding/json | 待写（第一波 1b） | 前置：encoding 包接口先行；具体类型子集——string/int/bool 标量、[]T、map[string]T、字段为上述类型的 struct 的 Marshal/Unmarshal 具体类型生成路径；any/interface{} 与通用反射式编解码不在第一波（须反射面，基底缺口）；float 类型待 float 落地后 | go_encoding_json_Marshal / Unmarshal |
| regexp | 待写（第一波 1b） | 前置：最小子集先行——Compile + Match/MatchString + FindString/FindAllString + ReplaceAllString，模式限字面量与基础元字符（. * + ? [...] ^ $ \|）；完整语法（分组、反向引用等）第二波 | go_regexp_* |
| slices | 待写（第一波 1b） | 前置：泛型实例化先行（前端泛型实例化，§8 缺口）；SLA 侧按具体类型实例化 | go_slices_* |
| maps | 待写（第一波 1b） | 前置：泛型实例化先行，同上 | go_maps_* |
| cmp | 待写（第一波 1b） | 前置：泛型实例化先行，同上；Compare/Or | go_cmp_* |
| sync/atomic | 待写（第一波 1b） | 前置：映射设计先行——SLA 原子操作面已存在（sa_std/sync/atomic：AtomicBool/I8/I16/I32/I64/U8/U16/U32/U64/Isize/Usize/Ptr + load/store/swap/add/CAS），今晚不须补基底；1b 工作是把 Go 指针式顶层函数（AddInt32/LoadInt64/CompareAndSwap 等）与 atomic.Int64/Uint64 等类型映射到该面 | go_sync_atomic_* |

#### 7.2.2 后续（84 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| archive/tar | 待写 | 须 io；纯归档解析/生成 | go_archive_tar_* |
| archive/zip | 待写 | 须 io + compress；纯 | go_archive_zip_* |
| compress/bzip2 | 待写 | 纯解压 | go_compress_bzip2_* |
| compress/flate | 待写 | 纯压缩 | go_compress_flate_* |
| compress/gzip | 待写 | 须 compress/flate；纯 | go_compress_gzip_* |
| compress/lzw | 待写 | 纯压缩 | go_compress_lzw_* |
| compress/zlib | 待写 | 须 compress/flate；纯 | go_compress_zlib_* |
| container/heap | 待写 | 须接口方法集；纯堆 | go_container_heap_* |
| container/list | 待写 | 纯双向链表 | go_container_list_* |
| container/ring | 待写 | 纯环形链表 | go_container_ring_* |
| crypto | 待写 | Hash/PublicKey 等接口定义 | go_crypto_* |
| crypto/aes | 待写 | 纯分组密码 | go_crypto_aes_* |
| crypto/cipher | 待写 | BlockMode/Stream 接口 | go_crypto_cipher_* |
| crypto/des | 待写 | 纯分组密码 | go_crypto_des_* |
| crypto/dsa | 待写 | 纯签名（须 math/big） | go_crypto_dsa_* |
| crypto/ecdh | 待写 | 纯曲线运算 | go_crypto_ecdh_* |
| crypto/ecdsa | 待写 | 纯签名（须 math/big） | go_crypto_ecdsa_* |
| crypto/ed25519 | 待写 | 纯签名 | go_crypto_ed25519_* |
| crypto/elliptic | 待写 | 纯曲线运算（须 math/big） | go_crypto_elliptic_* |
| crypto/hkdf | 待写 | 纯 KDF | go_crypto_hkdf_* |
| crypto/hmac | 待写 | 纯 MAC | go_crypto_hmac_* |
| crypto/hpke | 待写 | 纯混合加密 | go_crypto_hpke_* |
| crypto/md5 | 待写 | 纯摘要 | go_crypto_md5_* |
| crypto/mldsa | 待写 | 纯后量子签名 | go_crypto_mldsa_* |
| crypto/mlkem | 待写 | 纯后量子 KEM | go_crypto_mlkem_* |
| crypto/pbkdf2 | 待写 | 纯 KDF | go_crypto_pbkdf2_* |
| crypto/rc4 | 待写 | 纯流密码 | go_crypto_rc4_* |
| crypto/rsa | 待写 | 纯非对称（须 math/big） | go_crypto_rsa_* |
| crypto/sha1 | 待写 | 纯摘要 | go_crypto_sha1_* |
| crypto/sha256 | 待写 | 纯摘要 | go_crypto_sha256_* |
| crypto/sha3 | 待写 | 纯摘要 | go_crypto_sha3_* |
| crypto/sha512 | 待写 | 纯摘要 | go_crypto_sha512_* |
| crypto/subtle | 待写 | 常量时间比较，纯 | go_crypto_subtle_* |
| debug/dwarf | 待写 | 纯 DWARF 解析；低优先级 | go_debug_dwarf_* |
| debug/elf | 待写 | 纯 ELF 解析；低优先级 | go_debug_elf_* |
| debug/gosym | 待写 | 纯符号表解析；低优先级 | go_debug_gosym_* |
| debug/macho | 待写 | 纯 Mach-O 解析；低优先级 | go_debug_macho_* |
| debug/pe | 待写 | 纯 PE 解析；低优先级 | go_debug_pe_* |
| debug/plan9obj | 待写 | 纯 Plan 9 目标解析；低优先级 | go_debug_plan9obj_* |
| encoding | 待写 | BinaryMarshaler 等接口定义 | go_encoding_* |
| encoding/ascii85 | 待写 | 纯编码 | go_encoding_ascii85_* |
| encoding/asn1 | 待写 | 纯 DER 编解码 | go_encoding_asn1_* |
| encoding/base32 | 待写 | 纯编码 | go_encoding_base32_* |
| encoding/binary | 待写 | 纯二进制编解码 | go_encoding_binary_* |
| encoding/csv | 待写 | 须 io；纯 CSV | go_encoding_csv_* |
| encoding/json/jsontext | 待写 | json v2 流式 token 层；随 encoding/json | go_encoding_json_jsontext_* |
| encoding/json/v2 | 待写 | json v2；随 encoding/json | go_encoding_json_v2_* |
| encoding/pem | 待写 | 纯 PEM 编解码 | go_encoding_pem_* |
| encoding/xml | 待写 | 纯 XML 编解码 | go_encoding_xml_* |
| hash/adler32 | 待写 | 纯校验 | go_hash_adler32_* |
| hash/crc64 | 待写 | 纯校验 | go_hash_crc64_* |
| hash/maphash | 待写 | 纯哈希（须 seed） | go_hash_maphash_* |
| html | 待写 | 纯转义（EscapeString 等）；模板执行见 §7.3 | go_html_* |
| image | 待写 | 纯图像接口 + 解码注册表 | go_image_* |
| image/color | 待写 | 纯颜色模型 | go_image_color_* |
| image/color/palette | 待写 | 纯调色板 | go_image_color_palette_* |
| image/draw | 待写 | 纯绘制 | go_image_draw_* |
| image/gif | 待写 | 纯 GIF 编解码 | go_image_gif_* |
| image/jpeg | 待写 | 纯 JPEG 编解码 | go_image_jpeg_* |
| image/png | 待写 | 纯 PNG 编解码 | go_image_png_* |
| index/suffixarray | 待写 | 纯后缀数组 | go_index_suffixarray_* |
| io/fs | 待写 | FS 接口 + 内存 FS；OS 文件系统见 os（§7.3） | go_io_fs_* |
| io/ioutil | 待写 | 已废弃别名（ReadAll/ReadFile 等转 io/os）；随 io | go_io_ioutil_* |
| log/slog | 待写 | 须 io；结构化日志 | go_log_slog_* |
| math/big | 待写 | 大整数/大浮点，纯（体量大） | go_math_big_* |
| math/bits | 待写 | 纯位操作 | go_math_bits_* |
| math/cmplx | 待写 | 须 float 落地后 | go_math_cmplx_* |
| math/rand | 待写 | 纯伪随机（须 seed；真随机须 OS 熵，见 crypto/rand） | go_math_rand_* |
| math/rand/v2 | 待写 | 纯伪随机 v2 | go_math_rand_v2_* |
| mime | 待写 | 纯 MIME 类型判断 | go_mime_* |
| mime/multipart | 待写 | 须 io；纯 multipart | go_mime_multipart_* |
| mime/quotedprintable | 待写 | 须 io；纯编解码 | go_mime_quotedprintable_* |
| net/netip | 待写 | 纯 IP 解析（无 socket） | go_net_netip_* |
| net/url | 待写 | 纯 URL 解析（无网络） | go_net_url_* |
| path/filepath | 待写 | 纯路径操作（OS 分隔符按目标平台） | go_path_filepath_* |
| regexp/syntax | 待写 | 纯正则语法树 | go_regexp_syntax_* |
| testing/iotest | 待写 | 纯错误注入 Reader/Writer | go_testing_iotest_* |
| text/scanner | 待写 | 纯词法扫描 | go_text_scanner_* |
| text/tabwriter | 待写 | 须 io；纯制表 | go_text_tabwriter_* |
| text/template/parse | 待写 | 纯模板解析；执行见 §7.3（须反射） | go_text_template_parse_* |
| time/tzdata | 待写 | 须数据注入（embed 缺口，见 §7.3）；时区数据本身纯 | go_time_tzdata_* |
| unicode | 待写 | 纯分类表 | go_unicode_* |
| unicode/utf16 | 待写 | 纯 UTF-16 编解码 | go_unicode_utf16_* |
| uuid | 待写 | 纯 UUID 生成/解析（Go 1.27 新增） | go_uuid_* |

### 7.3 基底缺口-响亮拒绝（69 包）

#### 7.3.1 OS / 系统调用 / 网络（21 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| os | 基底缺口-响亮拒绝 | 须 OS 文件/进程/环境系统调用面，SLA 尚无 | —（响亮拒绝） |
| os/exec | 基底缺口-响亮拒绝 | 须进程创建面 | —（响亮拒绝） |
| os/signal | 基底缺口-响亮拒绝 | 须 OS 信号面 | —（响亮拒绝） |
| os/user | 基底缺口-响亮拒绝 | 须 OS 用户数据库面 | —（响亮拒绝） |
| syscall | 基底缺口-响亮拒绝 | 直通 OS 系统调用，SLA 尚无 | —（响亮拒绝） |
| plugin | 基底缺口-响亮拒绝 | 须动态加载面 | —（响亮拒绝） |
| log/syslog | 基底缺口-响亮拒绝 | 须 OS syslog 面 | —（响亮拒绝） |
| net | 基底缺口-响亮拒绝 | 须 socket 面 | —（响亮拒绝） |
| net/http | 基底缺口-响亮拒绝 | 须 net | —（响亮拒绝） |
| net/http/cgi | 基底缺口-响亮拒绝 | 须 net/http + os/exec | —（响亮拒绝） |
| net/http/cookiejar | 基底缺口-响亮拒绝 | 须 net/http | —（响亮拒绝） |
| net/http/fcgi | 基底缺口-响亮拒绝 | 须 net | —（响亮拒绝） |
| net/http/httptest | 基底缺口-响亮拒绝 | 须 net/http | —（响亮拒绝） |
| net/http/httptrace | 基底缺口-响亮拒绝 | 须 net/http | —（响亮拒绝） |
| net/http/httputil | 基底缺口-响亮拒绝 | 须 net/http | —（响亮拒绝） |
| net/http/pprof | 基底缺口-响亮拒绝 | 须 net/http + runtime | —（响亮拒绝） |
| net/mail | 基底缺口-响亮拒绝 | 须 net（SMTP 传输） | —（响亮拒绝） |
| net/rpc | 基底缺口-响亮拒绝 | 须 net + 编解码反射 | —（响亮拒绝） |
| net/rpc/jsonrpc | 基底缺口-响亮拒绝 | 须 net/rpc | —（响亮拒绝） |
| net/smtp | 基底缺口-响亮拒绝 | 须 net | —（响亮拒绝） |
| net/textproto | 基底缺口-响亮拒绝 | 须 net（文本协议传输） | —（响亮拒绝） |

#### 7.3.2 运行时 / 反射（13 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| runtime | 基底缺口-响亮拒绝 | 须运行时内省面（GC/栈/调度器） | —（响亮拒绝） |
| runtime/cgo | 基底缺口-响亮拒绝 | 须 cgo | —（响亮拒绝） |
| runtime/coverage | 基底缺口-响亮拒绝 | 须运行时覆盖率面 | —（响亮拒绝） |
| runtime/debug | 基底缺口-响亮拒绝 | 须运行时调试面 | —（响亮拒绝） |
| runtime/metrics | 基底缺口-响亮拒绝 | 须运行时指标面 | —（响亮拒绝） |
| runtime/pprof | 基底缺口-响亮拒绝 | 须运行时剖析面 | —（响亮拒绝） |
| runtime/race | 基底缺口-响亮拒绝 | 须竞态检测运行时 | —（响亮拒绝） |
| runtime/trace | 基底缺口-响亮拒绝 | 须运行时追踪面 | —（响亮拒绝） |
| reflect | 基底缺口-响亮拒绝 | 须运行时反射面 | —（响亮拒绝） |
| unique | 基底缺口-响亮拒绝 | 须运行时弱引用/驻留面 | —（响亮拒绝） |
| weak | 基底缺口-响亮拒绝 | 须运行时弱指针面 | —（响亮拒绝） |
| structs | 基底缺口-响亮拒绝 | 须编译器 struct 布局面（HostLayout 等） | —（响亮拒绝） |
| iter | 基底缺口-响亮拒绝 | 须闭包/函数值 + 泛型（前端缺口，§3/§5） | —（响亮拒绝） |

#### 7.3.3 工具链自举（14 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| go/ast | 基底缺口-响亮拒绝 | Go 工具链自举；用 gosla 编译工具链本身不在范围 | —（响亮拒绝） |
| go/build | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/build/constraint | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/constant | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/doc | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/doc/comment | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/format | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/importer | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/parser | 基底缺口-响亮拒绝 | 同上（注：gosla 宿主侧用 go/parser + go/types 作前端，不等于目标侧可用） | —（响亮拒绝） |
| go/printer | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/scanner | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/token | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |
| go/types | 基底缺口-响亮拒绝 | 同上（宿主侧用，不等于目标侧可用） | —（响亮拒绝） |
| go/version | 基底缺口-响亮拒绝 | 同上 | —（响亮拒绝） |

#### 7.3.4 模板 / 反射式 / 数据库 / 密码学（12 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| html/template | 基底缺口-响亮拒绝 | 模板执行须反射绑定数据 | —（响亮拒绝） |
| text/template | 基底缺口-响亮拒绝 | 同上（解析器见 §7.2.2 text/template/parse） | —（响亮拒绝） |
| encoding/gob | 基底缺口-响亮拒绝 | 任意类型编解码须反射 | —（响亮拒绝） |
| database/sql | 基底缺口-响亮拒绝 | 须驱动 + IO 面；无驱动则无意义 | —（响亮拒绝） |
| database/sql/driver | 基底缺口-响亮拒绝 | 须驱动面 | —（响亮拒绝） |
| expvar | 基底缺口-响亮拒绝 | 须 net/http 发布 | —（响亮拒绝） |
| crypto/tls | 基底缺口-响亮拒绝 | 须 net + OS 证书库 | —（响亮拒绝） |
| crypto/x509 | 基底缺口-响亮拒绝 | 校验须 OS 信任库；纯解析子集可后议 | —（响亮拒绝） |
| crypto/x509/pkix | 基底缺口-响亮拒绝 | 随 crypto/x509 | —（响亮拒绝） |
| crypto/rand | 基底缺口-响亮拒绝 | 须 OS 熵源；SLA 获熵源后可转待写 | —（响亮拒绝） |
| crypto/fips140 | 基底缺口-响亮拒绝 | FIPS 模块边界，须 FIPS 运行时 | —（响亮拒绝） |
| crypto/mlkem/mlkemtest | 基底缺口-响亮拒绝 | 测试辅助包，随测试面 | —（响亮拒绝） |

#### 7.3.5 其他（9 包）

| 包 | 状态 | 备注 | 映射 |
|---|---|---|---|
| embed | 基底缺口-响亮拒绝 | 须构建期文件嵌入面 | —（响亮拒绝） |
| debug/buildinfo | 基底缺口-响亮拒绝 | 须读取自身二进制 + 运行时构建信息面 | —（响亮拒绝） |
| testing | 基底缺口-响亮拒绝 | 测试 harness；gosla 编程序不编测试二进制 | —（响亮拒绝） |
| testing/cryptotest | 基底缺口-响亮拒绝 | 随测试面 | —（响亮拒绝） |
| testing/fstest | 基底缺口-响亮拒绝 | 随测试面（MapFS 本身纯，但无 harness 则无意义） | —（响亮拒绝） |
| testing/quick | 基底缺口-响亮拒绝 | 须反射 | —（响亮拒绝） |
| testing/slogtest | 基底缺口-响亮拒绝 | 随测试面 | —（响亮拒绝） |
| testing/synctest | 基底缺口-响亮拒绝 | 须 synctest 运行时 | —（响亮拒绝） |
| unsafe | 基底缺口-响亮拒绝 | v1 排除（§7.4）；unsafe 操作无 SLA 对应面 | —（响亮拒绝） |

### 7.4 包模型（3 项：缺口-前端 2 / 排除 1）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| 多文件包 / 自有包 | 缺口-前端 | 单文件 main 之外未做；import 模型（包路径解析、符号表）待结构线 | 待补 demo（拒收） |
| 导入别名 / 点导入 / 空白导入 | 缺口-前端 | import 声明整体被忽略，别名/点导入/空白导入均静默漏过；须显式拒收或接线 | 待补 demo（拒收） |
| unsafe / cgo | 排除 | v1 明确不做：unsafe 无 SLA 对应面，cgo 超出前端范围；遇上响亮拒绝 | 待补 demo（拒收） |


## 8. 接口 / 方法 / 泛型 / 错误（11 项：缺口-前端 11）

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| interface 类型声明 | 缺口-前端 | 无映射 | 待补 demo（拒收） |
| 方法定义（值/指针接收者） | 缺口-前端 | `funcDecl` 拒收 | 待补 demo（拒收） |
| 方法调用 x.M() | 缺口-前端 | 选择子调用拒收 | 待补 demo（拒收） |
| 方法集 / 嵌入 | 缺口-前端 | 无 | 待补 demo（拒收） |
| 类型断言 x.(T) | 缺口-前端 | 无处理分支 | 待补 demo（拒收） |
| 类型 switch | 缺口-前端 | 无 `TypeSwitchStmt` 分支 | 待补 demo（拒收） |
| any（空接口） | 缺口-前端 | 作结果/参数类型拒收；动态面归属待裁定（LANG-CANDIDATE-SLA-STD-DYN，见 §10） | 28_any（拒收） |
| 泛型（类型参数/约束/实例化） | 缺口-前端 | 无 | 待补 demo（拒收） |
| error 返回值惯例 | 缺口-前端 | 载体是多返回值（§5 缺口）+ error 接口（本节缺口）；两处通了才可做 | 待补 demo（拒收） |
| panic / recover | 缺口-前端 | 无处理分支 | 待补 demo（拒收） |
| errors.Is/As（错误链） | 缺口-前端 | 须 error 值先行；gostd 文本半 `go_errors_Is`、`go_errors_As` | 待补 demo |

## 9. 上游 SLA 面缺口与场地方差（7 项：部分 5 / 缺口-SLA 2）

> 本节不归 gosla：gosla 侧保持响亮拒绝，不得在 gosa 里打补丁绕。

| 项 | 状态 | 备注 | 验收 |
|---|---|---|---|
| 22_concat：SLA 无源级 Str concat 面 | 缺口-SLA | `sla_string_concat` 仅 ptr/len 句柄级；gosla 定位拒收 | 22_concat（拒收） |
| 23_strindex：SLA 无源级 Str 字节读取面 | 缺口-SLA | gosla 定位拒收 | 23_strindex（拒收） |
| Rust 路 SwitchExpr（07_switch） | 部分 | Zig 路绿；Rust 路 `UnsupportedSabDirectFeature: SwitchExpr`（上游缺口） | 07_switch（Zig 绿 / Rust 路待上游） |
| Rust 路 void+读参泄漏（08_funcs，缺陷 xxxiii） | 部分 | Zig 路绿；Rust 路 `error[MemoryLeak]`（上游缺陷队列） | 08_funcs（Zig 绿 / Rust 路待上游） |
| Rust 路 HashMap remove 解析（15_map） | 部分 | Zig 路绿；Rust 路 `TypeMismatch: remove receiver is not a Vec`（上游） | 15_map（Zig 绿 / Rust 路待上游） |
| Rust 路 string 字面量绑定（16_string，缺陷 xxv） | 部分 | Zig 路绿；Rust 路 `UnsupportedSabDirectFeature: string literal let binding`（上游） | 16_string（Zig 绿 / Rust 路待上游） |
| InstaBox 旧 std 无 Chan 面（27） | 部分 | 当前工具链（sci 79b716e5 + sa_plugin_sla 2db8411a）绿；旧 InstaBox host（0.1.9-era std）在 flattening 拒收（std 版本差，非发射缺陷） | 27_goroutine_chan（当前工具链绿） |

## 10. 汇总（2026-11-26，基线 c9fe2bd4，门禁 28/28）

- 语言语义面总数（§1–§6、§7.4、§8、§9 逐项）142 项：**支持 62、部分 11、
  缺口-前端 64、缺口-SLA 4、排除 1**。支持率 62/142 ≈ 44%，含部分 73/142 ≈ 51%。
  （修订：R1 +3 项（§3 新增 go/types 前置拒收，记支持）；R2 两项支持→部分
  （§4 for{}、§6 缓冲 chan）；总数 139 → 142。）
  分节：§1 程序结构 10 = 支持 4 / 部分 2 / 缺口-前端 4；
  §2 类型 17 = 支持 8 / 缺口-前端 9；
  §3 表达式 36 = 支持 21 / 部分 2 / 缺口-前端 11 / 缺口-SLA 2；
  §4 语句 31 = 支持 19 / 部分 1 / 缺口-前端 11；
  §5 函数 10 = 支持 4 / 缺口-前端 6；
  §6 并发 17 = 支持 6 / 部分 1 / 缺口-前端 10；
  §7.4 包模型 3 = 缺口-前端 2 / 排除 1；
  §8 接口/方法/泛型/错误 11 = 缺口-前端 11；
  §9 上游 SLA 面 7 = 部分 5 / 缺口-SLA 2。
- 标准库包总数（§7.1–§7.3，Go 1.27.2 公开包）180 包：**已写 8、待写 103
  （第一波·建议 19 = 1a 无前置 13 + 1b 有前置 6）、基底缺口-响亮拒绝 69**。已写率 8/180 ≈ 4%。
  分节：§7.1 已写 8；§7.2 待写 103 = 第一波 19 + 后续 84；
  §7.3 基底缺口 69 = OS/网络 21 + 运行时/反射 13 + 工具链自举 14 +
  模板/反射式/数据库/密码学 12 + 其他 9。
- 今晚 std 轨目标（DM 13239）：普查完成 + 第一波写完（1a 13 包优先；
  1b 6 包前置满足才进今晚，否则顺延第二波——见 §7.2.1）；
  验收口径不变（DM 13242）：demos/run.sh 全绿 + std 普查逐包状态，
  响亮拒绝面不降级。
- 转正依据：门禁复核 28/28（19 PASS + 9 REFUSED-OK，GATE_RC=0，
  见卷首；sa 0.1.11-优化 + Go 1.27.2 + 隔离 HOME + SA_STD_DIR）。
  18/18 发射 demo 的 `.sla` 逐字节比对通过；9/9 拒收 demo 的指纹全数命中。
- 已知静默缺口（集成前置阻塞项，须先行关闭——见 §11）：
  `init` 函数编出死代码、range over map 编出错误迭代、
  具名结果参数漏到 SLA 层报错、`close(ch)` 漏成未知调用、
  包限定值（除 `fmt.Println` 外）漏到 SLA 层报错、
  `nil` 直通未经端到端验证、可变参数 `...T` 当 slice 映射。
  以上 7 处在写手线代码落地前须由归属线补拒收守卫（或接线），
  否则集成门禁不得放行——静默错码是红线（AGENTS.md 规则 4）。
- 剩余真实缺口（按价值序）：defer、interface 与方法集（含 any/error 值惯例）、
  多返回值、包与导入模型、float 全系、方法调用/类型断言/类型 switch、
  标准库映射（180 包：已写 8、待写 103、基底缺口 69）、slice/map 复合字面量、
  位运算、指针、泛型、select default、close、
  顶层 var、多文件包。
- 登记上报（不归 gosla，勿自作主张）：
  LANG-CANDIDATE-SLA-STD-DYN（要不要给 sla_std 加动态面）待用户裁定，
  撞上就登记上报；22_concat/23_strindex 是 SLA 本体事，保持响亮拒绝。
- 历史读数（勿再引用为现值）：增量 1 时 16 绿（8 发射 + 8 拒收）；
  增量 2 时 28/28（18 发射 + 10 拒收）；增量 3 落后 28/28
  （19 发射 + 9 拒收，27_goroutine_chan 由拒收转发射）。

## 11. 增量计划 = 集成地图（四条 staging 线 + 两条 gostd 线）

> 用途变更（muse DM 13175 / DM 13185 / DM 13216，用户亲裁）：
> 不再逐增量推进；本计划同时充当四条 staging 线的集成地图——
> 哪份文件落进仓里哪个模块、接口点如何对齐、拒绝清单如何并入
> `expect.refused` 体系。线主小火车；Sunny 负责普查 + gostd 集成 + demo 核验。
> 前端沿 Go 本体继续（`go/parser` + `go/types`），裁定标准 = 速度；
> Rust 稿按 Go 模块布局 1:1 移植作设计稿，规则表与拒绝文案直接搬。
> 本地图按 Go 模块布局对齐（不按 Rust 布局）。
>
> 移交（muse DM 13239 / DM 13242，用户 15:53/15:54 亲令）：
> 主会话退出 gosla——自即刻起由 Sunny 和小火车合作完成全程
> （写、集成、调试、落地），主会话不再派写手线、不再下集成步骤。
> 本机 staging（`~/workspace/gosla-staging/`，六线已交五条、映射线收尾中）
> 全部移交我们，用不用、用多少由我们定。
> staging 文件目前在主会话本机，移交后对齐（落点见 11.1，到达后逐文件核对）。
> 验收口径不变：demos/run.sh 全绿 + std 普查逐包状态，响亮拒绝面不降级。

### 11.1 落点表（staging → Go 模块布局）

| staging 线 | 落点（Go 模块布局） | 承接的普查面 | 接口点 |
|---|---|---|---|
| 结构线 `staging/structure/` | `internal/gosla/gosla.go`（`Transpile` / `genDecl` / `funcDecl` / `block`）+ `cmd/gosla/main.go`（CLI 面若有改动） | §1 程序结构与声明、§7 多文件包/导入模型 | `Transpile` 入口、`genDecl` 的 Tok 分发、`funcDecl` 的签名处理；新结构代码（包/导入/init 接线）落为 `internal/gosla/` 内新文件，须经 `genDecl`/`funcDecl` 分发进入 |
| 类型线 `staging/types/` | `internal/gosla/gosla.go`（`slaType` / `zeroValue` / `constLiteral` / `fieldName`）+ `internal/gosla/emit_expr.go`（`compositeLit` / `indexExpr`） | §2 类型、§3 复合字面量/类型转换、§8 接口/any/泛型（类型侧） | `slaType` 是共享契约：新增类型须同时补 `slaType` 臂 + `zeroValue` 臂 +（有字面量时）`compositeLit` 臂，三者缺一不可；`fieldName` 首字母小写化是既定规则 |
| 语句线 `staging/statements/` | `internal/gosla/emit_stmt.go`（`stmt` / `assign` / `ifStmt` / `forStmt` / `rangeStmt` / `switchStmt`）+ `internal/gosla/emit_expr.go`（`expr` / `call`）+ `internal/gosla/emit_chan.go`（并发语句） | §3 表达式、§4 语句、§5 函数（语句侧）、§6 并发 | `stmt` 分发、`expr` 分发、`e.refuse` 定位拒收；循环/作用域须守 `loopDepth` 与 `rangeIncr`；并发语句须守 `emit_chan.go` 的生命期纪律（`liveGo` / 块末清理） |
| 标准库映射 + demo 语料线 `staging/stdlib/`、`staging/demos/` | `internal/gosla/emit_expr.go`（`call` 内新增分发臂）+ 新建 `internal/gosla/stdlib.go`（映射表：Go 调用 → `go_<pkg>_<name>` 签名 + 参数映射）+ `demos/NN_name/`（新 demo，编号自 29 起） | §7 标准库、§3 `fmt` 其余形态、§8 错误链 | `call` 的 `fmt.Println` 特判是既定形态，新映射臂照此办理；`Transpile` 的 hdr `@import` 块随新依赖增长；demo 形态见 11.3 |
| gostd 文本半（`~/workspace/gosla-staging/gostd/`：fmt/strings/strconv/errors） | SLA 侧函数 `go_<pkg>_<name>`；落点待定（见 11.2） | §7 fmt/strings/strconv/errors | 前端映射层按 `go_<pkg>_<name>` 签名对齐；Sunny 负责 gostd 集成 |
| gostd 运行时半（`~/workspace/gosla-staging/gostd/`：sync/time/sort/math） | SLA 侧函数 `go_<pkg>_<name>`；落点待定（见 11.2） | §7 sync/time/sort/math | 同上；math 全系 float，须 float 落地后才可接线 |

### 11.2 接口对齐点（四线共守）

1. **共享契约**：`emitter` 结构 + `Refusal`（定位、响亮）+ `slaType`（类型→SLA 映射）+
   `zeroValue` + `constLiteral`。新代码只许经这些契约扩展，不许另起映射表。
2. **拒绝文案直接搬**：Rust 稿的规则表与拒绝文案按 Go 模块布局 1:1 移植，
   文案原文搬运；`expect.refused` 的指纹行是 stderr 的子串，文案一经定稿不得随意改。
3. **不许静默错码**：任何未实现的形态必须走 `e.refuse`（AGENTS.md 规则 4）。
   §10 列出的 7 处已知静默缺口是集成前置阻塞项，归属线落地前必须补守卫：
   init→结构线；range over map→语句线；具名结果→结构线；
   close→语句线（或并发归属）；包限定值→类型线/语句线；
   nil→语句线（补 demo 或拒收）；可变参数→类型线。
4. **hdr `@import` 归属**：`Transpile` 的 hdr 块随新 `sa_std` 依赖增长，
   由引入依赖的线负责；gostd 的 `@import` 路径待落点定案后由 Sunny 接线。
5. **gostd 落点（待定，需裁定）**：`go_<pkg>_<name>` 函数的家——
   进 `sla_std` 还是作为 gosla 自带 runtime `.sla` 随发射器搭售——
   在集成时定；前端映射层只认签名，不假设落点。
   LANG-CANDIDATE-SLA-STD-DYN（动态面）是另一项待用户裁定的事，
   与 gostd 落点无关，不得混为一谈。
6. **生命期纪律**：并发代码须守 `emit_chan.go` 的块末 join/destroy、
   `liveGo()` 与 `loopDepth` 守卫；新并发形态不得绕过。
7. **只发 SLA 文本**：出口唯一；任何线不得直发 SA、不得手改生成的 `.sla`
   （AGENTS.md 规则 1）。

### 11.3 拒绝清单并入 expect.refused 体系

- 每个新增拒收形态配一个 demo 目录 `demos/NN_name/`：
  `main.go`（呈现该形态）+ `expect.refused`（指纹行 = 拒收 stderr 的子串，
  可多行；`#` 开头为注释行）。
- 每个新增支持形态配一个 demo 目录：`main.go` + `expected.stdout`
  （+ 由 emit 生成的 `main.sla`，提交前经门禁比对；`.sla` 是编译器输出，永不手改）。
- 编号自 29 起按落地顺序编；`demos/run.sh` 自动扫 `^[0-9]+_` 目录，无需改门禁脚本。
- 普查表「验收」列的「待补 demo（拒收）」即本条的待办清单；
  集成阶段每合入一条线，须把该线认领的「待补 demo」清掉（或注明仍缺口的原因）。

### 11.4 落地顺序与门禁

1. **集成顺序**（写并行、合串行）：类型线 → 结构线 → 语句线 → 标准库映射 + demo 语料线 →
   gostd 两半（文本半先、运行时半后；math 须 float 落地后）。
   类型线先行是因 `slaType` 是其余三线的共享契约。
2. **里程碑 spine → Part II**（R1294 投票建议 scope_staged 9–1，线主与 Sunny
   接受为工作顺序；决定权在两人）：先做单文件可用主干（spine：单文件 main +
   已有 28 个 demo 全绿为基线），随后立即接 Part II（import 图 + init 顺序）
   作为同一集成的下一步，不跳到全量。Part II 的 import 图与 init 接线都归结构线。
3. **每线合入检查点**：（a）staging 稿按 Go 模块布局移植到位；
   （b）认领的「待补 demo」已补（支持形配 `expected.stdout`，拒收形配 `expect.refused`）；
   （c）门禁全绿（基线 28/28，只能涨不能跌）；
   （d）任务分支上独立 commit。
4. **门禁调用**（本次普查实测有效，缺 `SA_STD_DIR` 会全败——见卷首）：
   `PATH=/opt/zig:/usr/local/go/bin:$PATH GOSLA_SA=/content/sa_all/sci/zig-out/bin/sa
   HOME=<隔离 HOME> SA_STD_DIR=/content/sa_all/sci/sa_std ./demos/run.sh`
   隔离 HOME 须含 sla 插件（现用 `/content/sunny-work/t162retest/home`，插件 sla 0.1.8）。
5. **分支纪律**：集成在任务分支上做（`task-170-census` 续用或另开
   `task-170-integration`，由线主定），不直接动 main，不推新远端分支；
   交付经主会话审核后合并。旧直发 SA 代码锁在 `legacy/go2sa-direct-sa`，不碰。
6. **验收清单**：本普查表即集成验收清单——每项的「验收」列是该项的门；
   集成完成的标准 = 普查表无「待补 demo」悬空（或悬空项均有登记的原因），
   且门禁全绿。
7. **审阅**：首稿审阅已完成（小火车 REVIEW-170-census.md，APPROVE WITH NOTES）；
   审阅意见已落回本文件修订（R1–R6），§10 重数已重算；修订版交小火车复核重数，
   确认后首报 muse。

## 审阅答复（小火车 REVIEW-170-census.md，APPROVE WITH NOTES，2026-11-26）

> 审阅文件 REVIEW-170-census.md（178 行，sha 29096217…）在小火车本机，
> 门禁重跑日志 gate-rerun-170.log（sha 23ae23a7…）；本修订按审阅意见
> （DM 13347 + 13350 全文）逐项处理。核实通过项（语言面重数、包面 180、
> 7 处静默缺口、门禁 28/28 独立重跑）均已确认，无改动。

### Q1：R1 的 3 个 go/types 前置拒收 demo——补进表，状态记支持

已补进 §3（新增“go/types 前置层拒收”3 项：09_typerefuse、24_strindex_oob、
25_strindex_store）。未划出普查口径——“普查表即集成验收清单”要求门禁的
9 个拒收 demo 全数入表，划出会使清单漏门。状态记**支持**：三者是 Go 本身的
编译期错误，正确行为就是定位拒收；`go/types` 层（gosla 前端的一部分）已实现
且 demo 端到端跑绿（REFUSED-OK），无后续工作。它们与 23_strindex（gosla
自身拒收，须 SLA 面）不是同一层，备注中已区分，不得互相覆盖。

### Q2：R6 拆 1a/1b——同意；sync/atomic、encoding/json、regexp 第一波范围

同意拆 1a（13 包，无前置）/1b（6 包，有前置），见 §7.2.1。
- **sync/atomic 的 SLA 原子操作面已存在**，今晚不须补基底：
  `sa_std/sync/atomic/atomic.sa`（1500 行）提供 AtomicBool/I8/I16/I32/I64/
  U8/U16/U32/U64/Isize/Usize/Ptr + load/store/swap/add/sub/and/or/xor/min/max/
  cmpxchg；1b 工作是把 Go 指针式顶层函数（AddInt32/LoadInt64/CompareAndSwap
  等）与 atomic.Int64/Uint64/Bool/Pointer 等类型映射到该面（映射设计先行）。
- **encoding/json 第一波具体类型子集**：string/int/bool 标量、[]T、
  map[string]T、字段为上述类型的 struct 的 Marshal/Unmarshal 具体类型生成路径；
  any/interface{} 与通用反射式编解码不在第一波（须反射面，基底缺口）；
  float 类型待 float 落地后。另须 encoding 包接口（BinaryMarshaler 等）先行。
- **regexp 第一波程度**：最小子集——Compile + Match/MatchString +
  FindString/FindAllString + ReplaceAllString，模式限字面量与基础元字符
  （. * + ? [...] ^ $ |）；完整语法（分组、反向引用等）第二波。

### Q3：R3 基线表述与 148 vs 139——已更正

卷首基线表述已更正（HEAD 是普查提交 a4503d5，未推远端；父提交才是远端头
c9fe2bd4）；提交信息“(148 items)”与文件实数不符——修订版文件实数为 142 项，
提交信息随修订版一并更正为 142 项。

### Q4：R5 spine→Part II 里程碑——接受；Part II 归结构线

接受。已入 §11.4 第 2 条：先单文件可用主干（spine），随后立即接 Part II
（import 图 + init 顺序）作为同一集成的下一步，不跳到全量；Part II 的
import 图与 init 接线都归结构线。

### Q5：staging 映射线何时移交到小火车本机——待 parent 确认

staging（`~/workspace/gosla-staging/`，六线已交五条、映射线收尾中）在主会话
本机，移交时间待 parent（Sunny）向 muse 确认后补告。本修订版先注“待确认”，
不编造时间。

## 附录：基线与复核记录

- 仓：`layola13/gosa`，远端头 `c9fe2bd4`（`gosla inc3: goroutines, channels
  and select onto SLA native surface`）；盒上副本 `/content/sunny-work/gosa`，
  分支 `task-170-census`（自 main 新开，未推远端）。
- 本地提交 `596ac27`：不在远端、不在盒上；以远端头为准开工。
- 门禁复核：2026-11-26 ~15:40 CST，28/28（19 PASS + 9 REFUSED-OK），
  `GATE_RC=0`；sa 0.1.11-优化，Go 1.27.2，sci 79b716e5，sa_plugin_sla 2db8411a。
- 插件版本写法：卷首“sla 插件 0.1.8”是隔离 HOME 内已安装插件 sap.json 所报
  版本；附录“sa_plugin_sla 2db8411a”是插件源码仓主分支头——两者指同一插件，
  前者是版本标识、后者是源码提交标识（审阅建议，已核对 sap.json）。
- 格式来源：sapy `SYNTAX-CENSUS.md`（2026-11-25，主会话；口径与状态定义照搬，
  状态值按 gosla 调整：缺口分「前端」/「SLA」两档）；psa 仓未取得
  （`layola13/psa` 需认证，不可读），理论取自 DM 13148 与 sapy 普查做法，
  不照抄实现。
- 本文件本地主本：`~/workspace/task170_gosla/CENSUS-170.md`；
  盒上副本：`/content/sunny-work/gosa/CENSUS-170.md`（任务分支）与
  `/content/sunny-work/delivery/task170/CENSUS-170.md`。
