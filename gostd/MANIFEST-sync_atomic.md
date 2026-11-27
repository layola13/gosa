# MANIFEST — gostd sync/atomic (task 170, wave 1b)

Writer line: sync/atomic package. Write-first, no compilation, no tests.
Oracle: Go 1.27.2 stdlib `src/sync/atomic/doc.go`, `doc_64.go`,
`type.go`, `value.go` (read 2026-11-27). The oracle directory also
contains `atomic.go`, which is a 14-byte download stub reading
"404: Not Found": in Go 1.27.2 there is no sync/atomic/atomic.go;
the primitives live in the runtime and are re-declared in doc.go /
doc_64.go. Nothing is mapped from atomic.go because there is
nothing in it.
Census: CENSUS-170.md rev1 §7.2.1 — sync/atomic, 1b, mapping design
first (sa_std atomic face already exists, no new base needed); Q2.
Splice position: sync_atomic goes AFTER log and context, at the END
of the splice order (errors, math, strconv, strings, fmt, sync,
time, sort, runtime, io, bytes, hash, crc32, fnv, base64, hex,
utf8, path, flag, bufio, log, context). Two notes on that order:
(1) the brief calls it a "21-file" order but lists 22 names, and
22 .sla files are on disk in wave1a + wave1a-sunny — the count is
22, not 21; (2) log.sla's header states a different internal order
(errors -> runtime -> sync -> time -> sort -> math -> strconv ->
strings -> fmt -> log). "After context, at the end" is unambiguous
under either ordering, and sync_atomic has no dependency that would
constrain it earlier, so the end position stands. Relative order
vs the other five 1b packages (encoding/json, regexp, slices, maps,
cmp — cmp/maps/regexp .sla files were already present in wave1b/
when this file was written) is unconstrained by sync_atomic: it
calls no symbol from any of them.

## Files

- VM: `~/workspace/task170-work/wave1b/sync_atomic.sla` — 1587 lines, 148 functions
- VM: `~/workspace/task170-work/wave1b/MANIFEST-sync_atomic.md` (this file)
- Box: not yet copied (write-first; no box work in this task)
- sha256 of `sync_atomic.sla`:
  c5c165da086df557f285035388ed39f11ea83626be630e83df2a0fb2d70bfc14

## Functions covered vs census scope

Census scope: the Go pointer-style top-level functions AND the
typed wrappers AND atomic.Value, all mapped onto the sa_std atomic
face. ALL covered (97 public + 51 private helpers = 148 functions).

Handle structs (9): `GoAtomicInt32`, `GoAtomicUint32`,
`GoAtomicInt64`, `GoAtomicUint64`, `GoAtomicUintptr`,
`GoAtomicPointer`, `GoAtomicBool` (one i64 cell each, v at offset
0), `GoAtomicValue` (kind cell + i/b/p payload), `GoAtomicValueResult`
(present + kind + i/b/p payload).

Kind constants: `GO_SYNC_ATOMIC_VALUE_KIND_EMPTY` (0),
`GO_SYNC_ATOMIC_VALUE_KIND_I64` (1),
`GO_SYNC_ATOMIC_VALUE_KIND_BYTES` (2),
`GO_SYNC_ATOMIC_VALUE_KIND_PTR` (3),
`GO_SYNC_ATOMIC_VALUE_KIND_IN_PROGRESS` (-1).

Constructors (8): `go_sync_atomic_new_int32`,
`go_sync_atomic_new_uint32`, `go_sync_atomic_new_int64`,
`go_sync_atomic_new_uint64`, `go_sync_atomic_new_uintptr`,
`go_sync_atomic_new_pointer`, `go_sync_atomic_new_bool`,
`go_sync_atomic_new_value`.

Top-level functions (39, doc.go + doc_64.go):
Int32/Uint32/Uintptr/Int64/Uint64 × (Swap, CompareAndSwap, Add,
And, Or, Load, Store) = 35, plus Pointer × (Swap,
CompareAndSwap, Load, Store) = 4. Add returns the NEW value;
And/Or/Swap return the OLD value; CompareAndSwap returns bool.
The 64-bit forms come from doc_64.go (build tag
`!(386 || arm || mips || mipsle)` — the 64-bit target set).

Typed wrappers (43, type.go): Bool × (Load, Store, Swap,
CompareAndSwap) = 4, Pointer × 4 = 4, Int32/Int64/Uint32/Uint64/
Uintptr × (Load, Store, Swap, CompareAndSwap, Add, And, Or) = 35.
The int/uint wrappers delegate to the top-level functions,
exactly like Go's type.go; Bool calls its helpers directly (Go
has no top-level bool functions); Pointer[T] is type-erased to
the i64-address Pointer functions.

atomic.Value (4, value.go): `go_sync_atomic_value_load`,
`go_sync_atomic_value_store`, `go_sync_atomic_value_swap`,
`go_sync_atomic_value_compare_and_swap`, plus 3 public payload
constructors for the frontend (`go_sync_atomic_value_i64`,
`go_sync_atomic_value_bytes`, `go_sync_atomic_value_ptr`).

Private helpers (51): `__go_sync_atomic_trunc_i32`,
`__go_sync_atomic_trunc_u32`, `__go_sync_atomic_bool01`,
per-width atomic helpers (i32 7, u32 7, i64 7, u64 7, uintptr 7,
pointer 4, bool 4 = 43 — load/store/swap/cmpxchg/fetch_add/
fetch_and/fetch_or, minus fetch_add/and/or for pointer and bool,
which Go does not provide), and Value helpers (5:
`__go_sync_atomic_value_empty`,
`__go_sync_atomic_value_wait_ready`,
`__go_sync_atomic_value_store_payload`,
`__go_sync_atomic_value_equal`,
`__go_sync_atomic_bytes_equal`).

Not in this file (out of census scope for sync/atomic): nothing
in the Go sync/atomic API is omitted. The deprecated pre-1.19
spellings are the same functions (no separate surface).

## Loud rejections + panic codes claimed

gostd block 77400–77499. Grep-verified free before use: no hit
for 77434, 77435, 900045 or 900046 in wave1a/, wave1a-sunny/,
census/, or drive/staging/gosla-staging/gostd/. Collisions: none
found.

| code  | where | meaning |
|-------|-------|---------|
| 77434 | Value Store/Swap/CompareAndSwap-new | Payload kind outside the supported set (i64 / Vec<u8> / ptr), including a nil payload (kind 0). Go panics on Store(nil), Swap(nil), CompareAndSwap(old, nil). One meaning: unsupported payload type. |
| 77435 | Value Store/Swap/CompareAndSwap | Type inconsistency: payload kind differs from the kind established by the first Store, or CompareAndSwap old/new kinds differ. One meaning: inconsistent concrete type. Covers all four Go inconsistency panics (Store, Swap, CAS-new-vs-stored, CAS-old-vs-new). |

No other function in this file panics. Nothing is a silent stub
(modulo the atomicity binding in the next section, which is a
documented placeholder, not a silent approximation).

Target ids: NONE claimed. sync/atomic has no error values in Go,
so the assigned 900045 / 900046 are LEFT UNCLAIMED; no error
sentinel is defined here. No TODO for extra codes/ids is needed:
two codes cover the two loud-refusal meanings.

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **THE ATOMIC BINDING (the big one).** Every atomic operation
   goes through one private helper per (width, op)
   (`__go_sync_atomic_i64_load` etc.); each helper body today is
   a PLAIN field access (correct values, NO atomicity, NO
   ordering) and each helper comment names the exact sa_std macro
   + ordering that replaces the body (seq_cst everywhere; acquire
   as the CAS failure ordering). Integration MUST replace every
   helper body with the named macro expansion (or a thin .sa shim
   exposing the macros as fns, if that is the bridge chosen) and
   confirm the cell layout (single-field structs, v at offset 0;
   little-endian x86_64; slot sizes AtomicI32 4 / AtomicI64 8 /
   AtomicUsize 8 / AtomicPtr 8 / AtomicBool 1) against the box
   atomic.sa + atomic.sal before this file is treated as atomic.
   Until then the file is single-threaded-correct only. The macro
   spellings were read from a LOCAL copy of atomic.sa
   (`~/workspace/task150-work/sci/sa_std/sync/atomic/atomic.sa`,
   md5 c7908241bcb4ed2567663acdac47c0ba, 1500 lines,
   byte-identical across four workspace copies: task32,
   audit-20261121, task137, task150), which matches the census box
   verification (1500 lines, AtomicBool/I64/U64/Ptr present,
   cmpxchg 49 hits) — but the mapping is still marked TO BE
   CONFIRMED AGAINST sa_std ON THE BOX AT INTEGRATION, because
   the box head may have moved since those copies were taken. No
   .sla file in the corpus calls EXPAND or touches these macros
   yet, so no .sla-level call spelling was invented; the import
   `@import "sa_std/sync/atomic.sa"` (the forwarding shim; the
   canonical file is `sa_std/sync/atomic/atomic.sa`) declares the
   dependency for the splice.
2. **Frontend materialisation of *T (a).** Go's top-level
   functions take *int32 etc.; SLA has no pointer to a scalar.
   The frontend materialises every Go variable ever passed to an
   atomic top-level function into a handle struct with the
   `go_sync_atomic_new_*` constructors, holds it for the
   variable's lifetime, and passes &handle. A raw Go *int32 with
   no handle is a frontend refusal. Handles must not be copied
   after first use (Go's rule for its typed wrappers).
3. **64-bit Add wrap (c).** Go's AddInt64/AddUint64/AddUintptr
   wrap mod 2^64; the bound macros wrap in hardware, but the
   placeholder bodies use plain i64 `+`. Whether SLA i64
   arithmetic wraps or traps on overflow is an OPEN QUESTION for
   the debug phase. The 32-bit Adds cannot overflow i64 and are
   exact either way.
4. **64-bit alignment (d).** Go's 32-bit-platform alignment
   caveat does not apply on the 64-bit x86_64 target (doc_64.go
   build tag); revisit if the target ever becomes 32-bit.
5. **Uintptr is 64-bit (e).** Uintptr rides ATOMIC_USIZE_* on
   this target; on a 32-bit target the binding would change to
   the U32 macros. Public functions take i64 either way.
6. **Pointer / unsafe seam (f).** unsafe.Pointer is an i64
   address (nil = 0); Pointer[T] is type-erased. The binding
   must convert i64 <-> native ptr at the helper boundary. A Go
   *T the frontend cannot reduce to an address is a frontend
   refusal.
7. **Value payload kinds (g).** Only three concrete kinds are
   supported (i64 / Vec<u8> / ptr); the frontend builds
   arguments with `go_sync_atomic_value_i64` / `_bytes` / `_ptr`
   and must refuse every other payload type (float, struct, bool,
   smaller ints, maps, chans...) at the mapping line, and must
   refuse nil payloads (77434 is the backstop). The holder
   cannot distinguish Go string from Go []byte (both Vec<u8>).
   First-Store kind is the Value's type forever (77435).

## Cross-file dependencies

- `sa_std/vec.sa` — Vec<u8> (Value payloads, byte comparison).
- `sa_std/sync/atomic.sa` — the macro face (import declares the
  dependency; no .sla-level call into it yet — seam 1).
- No gostd file is called: sync_atomic calls no symbol from
  sync.sla or any other gostd file (the dependency on sync.sla
  is conceptual only). No other gostd file calls sync_atomic
  yet. The other five 1b packages are independent of it.

## Divergences recorded (deliberate, in-file)

- Helper bodies are plain accesses, not atomic, until the
  integration binding lands (seam 1). The approximation is
  confined to the 51 helper bodies, each naming its macro.
- Value Swap / CompareAndSwap payload window: Go swaps the data
  pointer with one atomic SwapPointer; this file reads the old
  payload then stores the new one in two steps, so concurrent
  Swaps on one Value can interleave (kind word stays consistent).
  Single-threaded programs are unaffected; a true single-word
  swap would need the payload boxed behind one pointer, which
  needs a heap allocation surface the splice does not have.
- Value payload stores after the first Store are plain field
  stores (single-word for i64 / ptr kinds — atomic on x86_64,
  ordered by the kind word; multi-word for the Vec<u8> kind —
  the handle can tear under true concurrency). Go stores one
  pointer atomically.
- Value CompareAndSwap over a Vec<u8> payload does a deep byte
  comparison. Go compares with == on the concrete type: same
  result for a string payload, but Go PANICS for a []byte
  payload (slices are uncomparable) while gostd compares bytes
  and returns a boolean. The holder cannot distinguish string
  from []byte, so the deep comparison is the only uniform
  choice; the []byte panic is not reproduced.
- Go's atomic.Int32 etc. and atomic.Value must not be copied
  after first use; the same discipline applies to the handle
  structs and GoAtomicValue here.
- Go's Value first-store protocol (typ/data words, procPin
  spin) is modelled with the kind cell + in-progress marker
  (-1) + payload-then-publish ordering, without procPin
  (no preemption control exists in gostd).

## Verification

- NOT compiled, NOT tested (write-first batch, per task brief).
- sha256 of `sync_atomic.sla`:
  c5c165da086df557f285035388ed39f11ea83626be630e83df2a0fb2d70bfc14
  (1587 lines; 97 public + 51 private = 148 functions; no
  duplicate function names; every `__go_sync_atomic_*` reference
  resolves to a definition in the file; no `^||` closures;
  panic(77434) ×3, panic(77435) ×4, all in Value functions;
  no 9000xx number used except the header note that 900045 /
  900046 are unclaimed.)
- Oracle-verification status: doc.go, doc_64.go, type.go,
  value.go all read in full. atomic.go in the oracle directory
  is a 14-byte "404: Not Found" stub — verified by reading, not
  assumed. sa_std face: read from the local 1500-line copy
  (md5 above); NOT verified against the live box — that check is
  seam 1, for integration.
- TODOs left: none in the .sla file. Open questions carried
  forward (not TODOs in code): the i64 overflow wrap question
  (seam 3, debug phase); the .sla -> macro-layer bridge choice
  (seam 1, integration line); the splice-order count/order
  notes above (integration line to confirm the canonical order).

## Fix — wave 1b blocker A (2026-11-27, Sunny verification RESULT-170-verify-wave1b-r1394.md section 3.1)

The delivered file failed to parse: 32 if blocks used as statements
ended with a bare closing brace instead of brace-semicolon (SLA
requires the semicolon terminator; the parser stopped with
"found 'return', expected semicolon" at 28-file splice line 19608).
Mechanical fix only: exactly 32 lines changed, each "}" -> "};",
no other edit. Sites (sync_atomic.sla line, function): 482
__go_sync_atomic_trunc_i32, 495 __go_sync_atomic_bool01, 535
__go_sync_atomic_i32_cmpxchg, 591 __go_sync_atomic_u32_cmpxchg, 648
__go_sync_atomic_i64_cmpxchg, 707 __go_sync_atomic_u64_cmpxchg, 765
__go_sync_atomic_uintptr_cmpxchg, 824 __go_sync_atomic_ptr_cmpxchg,
858 __go_sync_atomic_bool_cmpxchg, 888/892
__go_sync_atomic_value_store_payload, 907/910/913/916/919
__go_sync_atomic_value_equal, 926/931 __go_sync_atomic_bytes_equal,
1439/1442/1445 go_sync_atomic_value_load, 1458/1467/1475
go_sync_atomic_value_store, 1492/1501/1509 go_sync_atomic_value_swap,
1528/1531/1541/1552/1556 go_sync_atomic_value_compare_and_swap.
Before sha: 3dfff0ddd1a11927106d6458cb84d32d70acb90fa8cbb13ea1dc716e7983ade0.
After sha: c5c165da086df557f285035388ed39f11ea83626be630e83df2a0fb2d70bfc14
(1587 lines, unchanged count). 28-file splice rebuilt: 27,952 lines,
sha 7830f7130de19b3f7430c159201e66978c92af8c6d397a1644a73a78ab54b072
(was f7ca3947...); sa check (fixed plugin) now parses past
sync_atomic and stops at "Undefined call: go_sort_vec" (defect (lii),
expected). Report: SYNC-ATOMIC-FIX-170.md.
