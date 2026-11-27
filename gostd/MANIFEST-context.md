# MANIFEST — gostd `context` (task 170, wave 1a)

- **File (VM):** `~/workspace/task170-work/wave1a/context.sla`
- **File (box):** `/content/xiaohuoche-work/task170/gostd/context.sla`
- **Lines:** 946 (incl. header contract ~230 lines)
- **SHA-256:** `773d7b73889e6057ea122b82ef05d2d1557fb842a3a3b9bb75a3582531500471`
- **Functions:** 33 (`^fn ` count: 21 public + 12 private `__go_context_*` / `__go_after_alloc`)
- **Oracle:** Go 1.27.2 stdlib `/usr/local/go/src/context/context.go` (807 lines, read from the box)
- **Census scope:** CENSUS-170.md rev1 §7.2.1, 1a no-prerequisite package `context`
  ("须 goroutine/chan 面（已通，§6）", mapping `go_context_*`)
- **Status:** write-first. NOT compiled, NOT tested (per task orders). No `sa`/`sla`
  invocation, no tests, no Kaggle, no push, no relay message, no other
  package touched.

## 1. Functions covered vs census scope

Task-enumerated scope — all covered:

| Go | gostd | Notes |
|---|---|---|
| `Background()` | `go_context_background() -> GoContext` | kind-0 root |
| `TODO()` | `go_context_todo() -> GoContext` | kind-0 root, `is_todo` flag kept for a future `String()` |
| `WithCancel(parent)` | `go_context_with_cancel(parent) -> GoContextWithCancel { ctx, cancel: GoCancelFunc }` | + `go_context_cancel(f)` |
| `WithCancelCause(parent)` | `go_context_with_cancel_cause(parent) -> GoContextWithCancelCause { ctx, cancel: GoCancelCauseFunc }` | + `go_context_cancel_with_cause(f, cause)` |
| `WithDeadline(parent, d)` | `go_context_with_deadline(parent, d: GoTime) -> GoContextWithCancel` | rides on `go_time_after`; parent-earlier-deadline shortcut + already-passed synchronous cancel, as Go |
| `WithTimeout(parent, t)` | `go_context_with_timeout(parent, timeout: i64) -> GoContextWithCancel` | `WithDeadline(Now().Add(t))`, as Go |
| `WithValue(parent, key, val)` | `go_context_with_value(parent, key_id: i64, val: GosVal) -> GoContext` | keys are frontend-assigned i64 ids; values are `GosVal` (fmt.sla) |
| `Context.Deadline()` | `go_context_deadline(c) -> GoContextDeadline { deadline: GoTime, ok: bool }` | first timer deadline up the chain |
| `Context.Done()` | `go_context_done(c) -> GoContextDone { ch: Chan<i64>, ok: bool }` | `ok == false` is Go's nil; the handed-out channel never fires (behaviorally identical) |
| `Context.Err()` | `go_context_err(c) -> ptr` | nil = live; stored sentinel ptr, same on every call |
| `Context.Value(key)` | `go_context_value(c, key_id: i64) -> GosVal` | tag-0 `GosVal` = absent |
| `CancelFunc` | `GoCancelFunc` handle + `go_context_cancel` | handle struct, not an fn value (see §4) |
| `CancelCauseFunc` | `GoCancelCauseFunc` handle + `go_context_cancel_with_cause` | same |
| `Cause(c)` | `go_context_cause(c) -> ptr` | nil when live; first-cancellation cause otherwise |
| `AfterFunc(ctx, f)` | `go_context_after_func(c, f: fn() -> void) -> GoAfterFuncStop` + `go_context_after_func_stop(s) -> bool` | f runs in its own goroutine; once-guarded stop |

Beyond the enumerated scope, also covered (same oracle file, trivial
extensions of the same machinery — no extra seams):

| Go | gostd | Notes |
|---|---|---|
| `WithDeadlineCause(parent, d, cause)` | `go_context_with_deadline_cause(parent, d: GoTime, cause: ptr) -> GoContextWithCancel` | `WithDeadline` is literally this with nil cause in the oracle |
| `WithTimeoutCause(parent, t, cause)` | `go_context_with_timeout_cause(parent, timeout: i64, cause: ptr) -> GoContextWithCancel` | same |
| `WithoutCancel(parent)` | `go_context_without_cancel(parent) -> GoContext` | kind-4 node: values pass through, cancellation stops |
| `Canceled` | `go_context_canceled() -> ptr` | target id 900031 |
| `DeadlineExceeded` | `go_context_deadline_exceeded() -> ptr` | target id 900032 |

## 2. Loud rejections + panic codes

Claimed in the gostd block 77400–77499 (checked against NOTES-runtime.md
whole-library map — no collision with 77401, 77410–77413, 77416, 77417,
77420, 77421, 77423, 77430–77439, 77440–77442, 77446, 77447, 77451, 77452):

| Code | Meaning | Where |
|---|---|---|
| **77418** | nil-context misuse: any public entry point given a `GoContext` (or `GoCancelFunc` / `GoCancelCauseFunc` / `GoAfterFuncStop` handle) whose node is null. Go panics on a nil parent (`cannot create context from nil parent`) and nil-pointer panics on a nil Context method call, so the loud failure matches. | `__go_context_check`, `go_context_cancel`, `go_context_cancel_with_cause`, `go_context_after_func_stop` |
| 77405 | **NOT OURS — belongs to bytes** (bytes.Buffer.Truncate / Next / Peek out of range). CORRECTION (R1378): an earlier draft claimed it stayed free; it does not. AfterFunc is implemented honestly for every context (own goroutine + once-guarded stop; no goroutine at all for never-cancelable ancestry), so context claims no code here. | — |

Target ids claimed in the gostd block 900000–999999 (checked against
NOTES-text.md: 900001/900002 strconv, 900010 io.EOF — no collision):

| Id | Sentinel | Message |
|---|---|---|
| **900031** | `context.Canceled` | `context canceled` |
| **900032** | `context.DeadlineExceeded` | `context deadline exceeded` |

Deliberately NOT a panic (Go semantics preserved, not refusals):
double cancel / double stop are first-wins no-ops (`false` from a
second `stop()`), exactly as Go specifies.

## 3. Open integration seams (frontend / debug phase)

1. **(a) Done channel** — cancellation closes the node's `Chan<i64>`
   (`Chan::new(0)` + `.close()`, the `sa_std/ctx.sla` broadcast idiom;
   closing wakes every receiver/select arm at once). `Done()` returns
   `GoContextDone { ch, ok }` because SLA has no nil channel literal;
   `ok == false` means Go's nil and the handed-out channel never fires.
   Frontend must check `ok` only where Go code tests `Done() == nil`.
2. **(b) Deadline / GoTime** — deadlines are `GoTime` (time.sla);
   `WithTimeout` uses `go_time_now` + `go_time_add`; the fire channel
   comes from `go_time_after(dur)`. No timer behaviour is faked: if
   the parent's deadline is earlier, no timer is armed at all.
3. **(c) Value keys are i64 ids** assigned by the frontend (one id per
   Go key value; never reuse an id for two keys). Go's nil-key /
   non-comparable-key panics cannot occur. Values ride as `GosVal`
   (fmt.sla); a miss returns tag-0 `GosVal`.
4. **(d) AfterFunc goroutine** — `f: fn() -> void` is captured by the
   spawned closure and run there (IC-2). The stop handle is a struct,
   not an fn value (see §4).
5. **(e) WithValue parent-chain lookup** — `Value`/`Deadline`/`Done`/
   `Err`/`Cause` walk the `parent` chain; a `WithoutCancel` node cuts
   the cancellation walk but not the value walk. No
   `Value(&cancelCtxKey)` trick is needed (parent links are walked
   directly).
6. **CancelFunc / CancelCauseFunc / AfterFunc-stop are handle structs**
   (`GoCancelFunc` / `GoCancelCauseFunc` / `GoAfterFuncStop`, each one
   node ptr). The frontend emits `go_context_cancel` /
   `go_context_cancel_with_cause` / `go_context_after_func_stop` for
   the Go call. Rationale: SLA has no proven spelling for RETURNING
   an fn from a function (fn types are proven only in parameter
   position: `sync.Once.Do`, sort cores).
7. **Sentinel identity** — `ctx.Err()` returns the SAME stored ptr on
   every call for one context (allocated once at cancel time). A fresh
   `go_context_canceled()` call allocates a new node with the same
   target id, exactly like `go_strings_eof()` does for io.EOF, so
   pointer equality against a freshly built sentinel does NOT hold;
   the frontend must compare through `errors.Is` (target id).
8. **Splice position: AFTER fmt.sla (last).** Order: errors → runtime
   → sync → time → sort → math → strconv → strings → fmt → context.
   The brief said "after sync/time"; the GosVal dependency (values
   ride as `GosVal`) forces after fmt. Only fmt dependency is `GosVal`
   + `go_val_nil`. If `GosVal` ever moves earlier, context can move
   up with it. Other cross-file deps: `GoMutex` + `go_sync_*` (sync),
   `GoTime` + `go_time_now/add/after` (time), `go_errors_*` (errors).
9. **No `String()` methods** — `Background`/`TODO`/`WithCancel`/
   `WithDeadline`/`WithValue` name strings need fmt + Go type names,
   which gostd does not carry. There is no such function to call (not
   a silent stub). The `is_todo` flag preserves the
   Background-vs-TODO distinction for a future implementation.
10. **No foreign Context implementations** — gostd contexts are always
    `GoCtxNode` values, so Go's fallbacks for custom parents (the
    watcher goroutine in `propagateCancel`, the `afterFuncer` path)
    never arise. A frontend handing gostd a context built any other
    way has no constructor to call, so this cannot happen by accident.

## 4. Deliberate divergences (documented in-file, none silent)

- Children kept in `Vec<ptr>` with linear scans, not Go's
  `map[canceler]struct{}`. Same semantics; registration order is
  creation order (Go leaves it unspecified).
- Cancel holds at most one node lock at a time (snapshot children,
  unlock, then cancel them). Go notes it acquires the child's lock
  while holding the parent's; gostd's order is observationally
  identical and deadlock-free by construction.
- One extra goroutine per timer context (deadline watcher) and per
  `AfterFunc` registration; both exit promptly on manual cancel /
  stop via `select` (Go stops the runtime timer instead). No goroutine
  is spawned for `AfterFunc` on never-cancelable ancestry.
- `Done()` on a never-cancelable context hands out the node's own
  never-closed channel with `ok == false` (see §3.1).
- `GoContext` is a heap node behind `{ node: ptr }`; copying aliases
  the node (Go interface-value semantics). The node is never freed
  (same as `GosError`: gostd never frees; ownership belongs to the
  allocating program).

## 5. Integration checks for the debug phase (also in-file, IC-1–IC-10)

- **IC-1** `select` inside a `thread::spawn` closure (deadline watcher
  + AfterFunc waiter). Fallback if it fails: blocking `recv` on the
  fire/done channel; the watcher then lingers until the deadline /
  cancellation.
- **IC-2** calling a CAPTURED `fn() -> void` value inside a spawned
  closure (`AfterFunc` runs `f()`).
- **IC-3** `Box::new` over a struct containing `Chan` + `GoMutex` +
  `GosVal` + `Vec` fields (`GoCtxNode`, `GoAfterNode`).
- **IC-4** `&(*n).mu` — borrowing a struct field through a raw-ptr deref.
- **IC-5** `GoTime` literal construction + field reads from this file.
- **IC-6** `GosVal` as a struct field + `go_val_nil()` in a struct literal.
- **IC-7** Splice order: context AFTER fmt.sla (§3.8).
- **IC-8** `select` arms binding two values with underscore names.
- **IC-9** `Chan::new(0)` + `.close()` broadcast wakes every select arm.
- **IC-10** `AfterFunc` stop-vs-fire once-guard race (verify with a race demo).

## 6. What is NOT here

- No `String()` methods (§3.9). No compilation, no tests (write-first).
- No changes to any other file: the already-written 8 packages
  (errors/strconv/strings/fmt/sync/time/sort/math), `runtime.sla`,
  Sunny's files/repo, and the census were not touched. Nothing pushed.
- The census §7.2.1 row for `context` stays "待写（第一波 1a）" until
  integration + demo lands it; this file is the SLA-side write only.

## 7. Public API index (21 public functions)

`go_context_background`, `go_context_todo`, `go_context_with_cancel`,
`go_context_cancel`, `go_context_with_cancel_cause`,
`go_context_cancel_with_cause`, `go_context_with_deadline_cause`,
`go_context_with_deadline`, `go_context_with_timeout_cause`,
`go_context_with_timeout`, `go_context_with_value`,
`go_context_without_cancel`, `go_context_deadline`, `go_context_done`,
`go_context_err`, `go_context_value`, `go_context_cause`,
`go_context_after_func`, `go_context_after_func_stop`,
`go_context_canceled`, `go_context_deadline_exceeded`.
Plus result/handle structs: `GoContext`, `GoCtxNode`,
`GoContextDeadline`, `GoContextDone`, `GoContextWithCancel`,
`GoContextWithCancelCause`, `GoCancelFunc`, `GoCancelCauseFunc`,
`GoAfterFuncStop`, `GoAfterNode`.

## 8. CONTEXT-B-FIX-170 (2026-11-27, Muse relays 16738/16740) — blocker B type-declaration fix

- Design ruling 16740: blocker B is a LIBRARY-side type-declaration
  defect, not a checker gap. `__go_context_alloc_node` declared
  return `*GosError` but returned `*GoCtxNode` via `Box::into_raw`;
  `__go_after_alloc` likewise (`*GoAfterNode`). The checker was
  right to reject (TypeMismatch at splice 15027/15047 — §6 layer 4
  of SLA-FIXES-170.md closes with this fix; the planned post-(lii)
  bisect is cancelled).
- Changed (23 declaration/call-site edits + 3 move-discipline fixes,
  all asserted count==1): node-carrying fields, params and locals
  retyped `ptr` → `*GoCtxNode` / `*GoAfterNode` (`GoContext.node`,
  `GoCtxNode.parent` / `registered_on`, `GoCtxNode.children` →
  `Vec<*GoCtxNode>`, `GoCancelFunc.node`, `GoCancelCauseFunc.node`,
  `GoAfterFuncStop.node`, the two alloc fns, `__go_context_nil()` →
  `*GoCtxNode` (`0 as *GoCtxNode`), new `__go_after_nil()` →
  `*GoAfterNode`, cancel / remove / register / arm helpers);
  `err_ptr` / `cause` struct-literal initialisers now use
  `go_errors_nil()`. Three UseAfterMove layers surfaced as the
  checker advanced past the declaration fix, each fixed in-library
  (bufio-flush class: field stores move the RHS local): `err_ptr`
  double-construction in `__go_context_cancel_node` (same target
  id, errors.Is comparison per the S9 rule); `(*n).cause` read back
  as the recursive-call arg; unlock-before-`registered_on`-store in
  `__go_context_register_child` (safe: the child handle does not
  escape until return; the parent's children list is already
  updated under the lock). Probes: `0 as *GoCtxNode` return,
  `*GoCtxNode` struct field, `(*n).kind` deref, `==` nil-helper
  comparison — all check rc=0.
- New sha (VM and box):
  `22b6f1204c50e1240222637f3ebd52bf2ebaaab812d0b6c902290cda59c5fb41`
  (982 lines; was `97e68633…`, 959 lines). Public API unchanged
  (21 functions, §7); one new private helper `__go_after_nil()`.
- Check-level verification: 28-file diagnostic splice check reaches
  the slices cmp defect (CONTEXT-B-FIX-170.md §5); 24-file mini-splice
  (22 + cmp + maps, Chan sites neutralised in scratch only) rc=0.
  Gate 28/28 GREEN (19 PASS + 9 REFUSED-OK).
- Newly surfaced pre-existing defect (not this fix): `Chan::new`
  in struct-literal / tuple-return positions (`sync.sla:58/:289`,
  `time.sla:688/:758`, `context.sla:399/:416`) fails the checker's
  let-binding rule; needs a Muse ruling (library-side rewrite vs
  checker relaxation). See CONTEXT-B-FIX-170.md §5.
- Full report: CONTEXT-B-FIX-170.md (local ~/workspace/task170-work/).
