# MANIFEST — gostd slices (task 170, wave 1b)

Writer line: slices package. Write-first, no compilation, no tests,
no box work, no pushes (per task brief).
Oracle: Go 1.27.2 stdlib `src/slices` (read 2026-11-27):
`slices.go` (531 lines), `sort.go` (196), `iter.go` (114),
`zsortanyfunc.go` (479), `zsortordered.go` (481).
Census: CENSUS-170.md rev1 §7.2.1 — slices, 1b with the
generic-instantiation prerequisite, handled SLA-side by
concrete-type instantiation (see the file header).

## Files

- VM: `~/workspace/task170-work/wave1b/slices.sla` — 2005 lines,
  129 functions (96 public + 33 private helpers), 1 struct
  (`GoSlicesSearchResult`)
- VM: `~/workspace/task170-work/wave1b/MANIFEST-slices.md` (this file)
- sha256 of `slices.sla`:
  b9e989dfdec7618e1154a6ff234c50e61a9e2e63f71f972f0dd0e9bba60b35e9
  (this manifest's own hash is not recorded inside itself.)

## Functions covered vs census scope

Census §7.2.1 names no function-level list for slices — just the
generic-instantiation prerequisite — so the brief's explicit
function list governs. ALL 32 listed functions are covered, in
all three instantiations (_i64 / _bytes / _str = 96 public
functions). The three instantiations:

| suffix | Go type | SLA type |
|--------|---------|----------|
| _i64 | []int / []int64 | Vec<i64> |
| _bytes | []byte | Vec<u8> (element u8) |
| _str | []string | Vec<Vec<u8>> (each string a Vec<u8>) |

Per instantiation (32): Equal, EqualFunc, Compare, CompareFunc,
Index, IndexFunc, Contains, ContainsFunc, Insert, Delete,
DeleteFunc, Replace, Clone, Compact, CompactFunc, Grow, Clip,
Concat, Repeat, Reverse, Sort, SortFunc, SortStableFunc,
IsSorted, IsSortedFunc, Min, Max, MinFunc, MaxFunc,
BinarySearch, BinarySearchFunc, Chunk.

Private helpers (33): `__go_slices_log2`,
`__go_slices_cmp_i64`, `__go_slices_cmp_u8`,
`__go_slices_less_i64`, `__go_slices_less_u8`,
`__go_slices_less_str`, plus per type
`__go_slices_append_<t>`, `__go_slices_swap_<t>`,
`__go_slices_insertion_cmp_<t>`,
`__go_slices_sift_down_cmp_<t>`,
`__go_slices_heap_cmp_<t>`,
`__go_slices_median3_cmp_<t>`,
`__go_slices_quick_cmp_<t>`,
`__go_slices_merge_cmp_<t>`,
`__go_slices_stable_rec_cmp_<t>`.

Notes on the list:

- **Repeat** was NOT in the brief's function list, but it IS in
  the `slices.go` oracle. It is included (trivial, 3 short
  functions) rather than silently dropped; if the coordinator
  wants the file to match the brief's list exactly, delete the
  three `go_slices_repeat_<t>` functions.
- **Chunk** is covered, but EAGER (see seam (d) below): Go's
  Chunk returns `iter.Seq[Slice]`; this file returns
  `Vec<Vec<T>>` with all chunks collected. Same elements,
  order, and n < 1 panic; laziness is the divergence.
- **Sort / SortFunc / SortStableFunc / IsSorted /
  IsSortedFunc / Min / Max / MinFunc / MaxFunc /
  BinarySearch / BinarySearchFunc** (the `sort.go` surface)
  are all covered in all three instantiations.
- **OUT — no functions in this file (iter.go iterator
  surface, loud refusal, seam (e)):** All, Backward, Values,
  AppendSeq, Collect, Sorted, SortedFunc, SortedStableFunc.
  Every one takes or returns an `iter.Seq` value, which has
  no SLA spelling. The frontend must refuse them at the
  call site; there is deliberately no stub for any of them
  in this file.

## Loud rejections + panic codes claimed

gostd block 77400–77499.

| code | where | meaning |
|------|-------|---------|
| 77429 | Insert (i < 0, i > len), Delete/Replace (i < 0, j < i, j > len), Grow (n < 0), Chunk (n < 1), Min/Max/MinFunc/MaxFunc (empty slice), Concat/Repeat (length overflow) — 51 panic sites total | Slices bounds backstop: every genuine Go panic on inputs in this file. Go panics with a message in each case; SLA panic() takes only a code, so the code is the whole signal. |

**77430 — NOT claimed (collision found, TODO for the
coordinator).** The brief assigned 77430 for the
iterator/range-over-func loud refusal, but grep over
`wave1a/`, `wave1a-sunny/`, `census/`, `drive/`,
`staging/` shows 77430 falls inside sort's reserved block
77430–77439 (`sort.sla` header, `log.sla` header,
`NOTES-runtime.md`: "no codes used"). fmt renumbered off
77430/77431 for exactly this reason (`fmt.sla:14`,
`NOTES-text.md:113-115`). Per the brief's collision rule
the code is NOT used here. The iterator loud refusal
(All/Backward/Values/AppendSeq/Collect/Sorted*) needs a
fresh code allocated by the coordinator, outside sort's
block; until then the refusal is frontend-side with no
gostd code. No other 774xx code is used by this file.

**Target ids 900040 / 900041 — NOT claimed.** Grep shows
both are free (no hits anywhere in the task tree), but
slices has no error values in Go — no sentinel, no
(T, error) result — so neither id is claimed. Do not
invent sentinels for them.

## Collisions found (grep-verified)

- 77429: free (no hits in wave1a, wave1a-sunny, census,
  drive, staging) — claimed.
- 77430: COLLISION with sort's reserved block 77430–77439
  (see above) — NOT used, TODO.
- 900040 / 900041: free, but unclaimed (no error values).
- Function names: no duplicates inside `slices.sla`, and no
  collisions with any `^fn` name in `wave1a/*.sla` or
  `wave1a-sunny/gostd/*.sla` (verified by comm on sorted
  name lists).

## Splice position + cross-file dependencies

Splice position: **AFTER context — 23rd file**, last in the
IO-CONTRACT-170.md §B-S10 order (errors, math, strconv,
strings, fmt, sync, time, sort, runtime, io, bytes, hash,
crc32, fnv, base64, hex, utf8, path, flag, bufio, log,
context, **slices**). slices depends only on sort (8th) and
bytes (11th), both earlier, so it goes last. No dependency
on the cmp package (also wave 1b, unwritten): comparison
logic here is local (i64/u8 operators, `bytes.sla`).

Cross-file dependencies (all public APIs in earlier files):
- `sort.sla`: `go_sort_vec` (Sort_i64/_bytes/_str).
- `bytes.sla`: `go_bytes_compare` (string element compare /
  ordering), `go_bytes_clone` (string element copies).
- No other cross-file calls. The file includes
  `@import "sa_std/vec.sa"` like `sort.sla` does (the
  splicer strips duplicate imports).

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **Comparators are fn-value parameters (a).** Every *Func
   variant takes `eq`/`del`/`f` (`fn(&T)->bool`,
   `fn(&T,&T)->bool`) or `cmp` (`fn(&T,&T)->i64`, -1/0/+1).
   The frontend MUST emit a named top-level fn per Go func
   value (e.g. `__go_slices_cmp_i64` per call site) and pass
   it; a `^||` closure literal at the call site is rejected
   by the checker in every position (R1400) and must be
   hoisted to a named fn first. This mirrors sort.sla's
   `less: fn(&T, &T) -> bool` precedent. The tag-dispatch
   alternative (an i64 ordering-code parameter) was NOT
   chosen: it cannot express arbitrary user comparators.
2. **Generic instantiation (b).** Only i64 / u8 / Vec<u8>
   element types exist here. Any other T needs a new
   instantiation; the frontend must not silently route
   another T through these three.
3. **Variadic packing (c).** Insert/Replace take v as one
   `Vec<T>`; Concat takes `Vec<Vec<T>>`. The frontend packs
   the variadic arguments first.
4. **Chunk is eager (d).** Returns `Vec<Vec<T>>`, not an
   iterator. Same chunks, no laziness.
5. **Iterators are OUT (e).** All/Backward/Values/AppendSeq/
   Collect/Sorted/SortedFunc/SortedStableFunc must be
   refused by the frontend; code TODO (77430, above).
6. **Fresh-Vec length ops (f).** Insert/Delete/DeleteFunc/
   Replace/Clone/Compact/CompactFunc/Grow/Clip/Concat/
   Repeat/Chunk never mutate their input; they return a
   fresh Vec. A second header aliasing the same buffer
   does not observe the change (Go would show it). The
   frontend must rebind every live header to the returned
   value. Reverse/Sort/SortFunc/SortStableFunc DO mutate
   in place and take `&Vec<T>`.
7. **Sort algorithm (g).** Sort reuses sort.sla's introsort;
   SortFunc/SortStableFunc use cmp-threaded transcriptions
   of the same introsort/merge shapes (NOT Go's pdqsort).
   Same resulting order for a strict weak ordering.
8. **BinarySearchFunc E == T (h).** Go allows a different
   target type; this instantiation keeps element and target
   the same type.
9. **Copy rule (i).** `Vec::push` moves its argument; plain
   fn args do not consume Vec values. String elements are
   copied only via `go_bytes_clone` (never pushed out of a
   live slice); Min/Max/MinFunc/MaxFunc for _str track the
   best index and return a clone; the _str stable sort
   builds its scratch from clones. A deep copy of a string
   element is unobservable (Go strings are immutable).

## Divergences recorded (deliberate, in-file)

- nil vs empty not distinguished (no nil Vec in gostd).
- Capacity not modelled; Grow/Clip are no-ops (Grow still
  panics on n < 0).
- Length-changing ops return fresh Vecs; aliases do not
  observe the change (seam 6).
- Chunk is eager, not lazy (seam 4).
- Sort is introsort, not pdqsort (seam 7); same order for a
  strict weak ordering.
- Repeat included although not in the brief's list (see
  above); Concat/Repeat overflow panics use 77429.
- BinarySearch uses `i + (j - i) / 2` (sort.sla's
  `go_sort_search` shape) instead of Go's `uint(i+j) >> 1`;
  identical results at any reachable length.
- Concat's overflow check catches a total in [2^63, 2^64)
  per add; a total >= 2^64 is unreachable (it would need
  > 2^64 elements).

## Oracle-verification status

- Read, all five oracle files: `slices.go` (531 lines),
  `sort.go` (196), `iter.go` (114), `zsortanyfunc.go` (479),
  `zsortordered.go` (481). `zsortordered.go`/`zsortanyfunc.go`
  (pdqsort) were read to record the algorithm divergence;
  pdqsort itself was NOT transcribed (seam 7).
- Could NOT verify (write-first, no box work per brief):
  no `sa check`, no build, no tests. In particular the
  following shapes were chosen to match proven corpus
  precedent, but are unverified in this file until the
  splice check runs:
  - fn-value comparator parameters (`fn(&T,&T)->i64`) —
    same shape as sort.sla's `less: fn(&T,&T)->bool`.
  - `go_sort_vec(s, __go_slices_less_<t>)` — same shape as
    `go_sort_ints` / `go_sort_strings`.
  - `scratch.push(go_bytes_clone(s[i]))` for _str stable
    sort (avoids the unproven push-then-read shape that
    `go_sort_stable`'s `scratch.push(v[i])` would have
    for Vec<u8> elements).
  - the _str merge's read-once/write-back slot discipline
    (same shape as sort.sla's swap/merge).
  - `go_bytes_compare(s[i], v)` with v a by-value Vec<u8>
    parameter used across loop iterations (fn args do not
    consume, per R1400).
- Style precedents read before writing: `wave1a/log.sla`
  (full file), `wave1a-sunny/gostd-live-io.sla` (header),
  `wave1a/MANIFEST-log.md`, `wave1a-sunny/gostd/sort.sla`
  (full file), `wave1a-sunny/gostd/bytes.sla` (idioms:
  `__go_b_append`, `__go_b_slice`, `GoBytesCut` structs,
  `go_bytes_clone`/`go_bytes_compare`, `!` negation,
  rebuild-to-shorten). No `^|| {` block closures anywhere
  in the file (grep-verified); no closures at all.

## Verification

- NOT compiled, NOT tested (write-first batch, per task brief).
- sha256 of `slices.sla` (VM):
  b9e989dfdec7618e1154a6ff234c50e61a9e2e63f71f972f0dd0e9bba60b35e9
  (2005 lines; 96 public + 33 private = 129 functions;
  51 panic(77429) sites; 1 struct).
