# gosa — go→sa frontend (`go2sa`)

Go 语言新增编译目标 `go2sa`：`.go` → `.sa`（`sci/sa` 消费）。

参考 `tsgosa`（`ts→sa`）链路与门禁，前端换 `go/parser` + `go/types`。

```
.go --[parser+types]--> analyze --[desugar]--> middle --[emit SA + sci macros]--> .sa --> [sci sa build] --> .exe/.wasm
```

- Go 源码浅克隆于 `/content/go`（只读移植源，不进本仓）。
- CLI：`go2sa --sa [--out <dir>] <file.go> [...]`（exit 2 用法/IO，1 拒收，0 通过）。
- 门禁：`demos/run.sh`（`--check` 逐字节比对，`.sa` 禁止手改）。

状态：S1 skeleton（parse only，lowering 未实现，一律定位拒收）。
