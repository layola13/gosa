# 27_goroutine_chan

Go goroutines + unbuffered channels + select, emitted onto SLA's
native surface: `Chan::new(0)`, `ch.send(v)`, `thread::spawn(^|| f(...))`,
and an SLA `select` with `case let (v, closed) = ch.recv()` arms.
Function/block-end cleanup: spawned handles are joined and locally
made channels destroyed at the end of the creating block.

E2E venue note: verified on the current toolchain (sci 79b716e5 +
sa_plugin_sla 2db8411a): prints 338350, rc=0. The InstaBox sci-base
host (0.1.9-era std) predates the Chan surface in signatures and
rejects this file at flattening (ForbiddenSyntax on the fn header) —
std version skew, not an emission defect.
