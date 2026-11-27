# MANIFEST — gostd cmp (task 170, wave 1b)

Writer line: cmp package. Write-first, no compilation, no tests.
Oracle: Go 1.27.2 stdlib `src/cmp/cmp.go` (70 lines, the whole
package is one file), read 2026-11-27.
Census: CENSUS-170.md rev1 §7.2.1 — cmp, 1b package with the
generic-instantiation prerequisite, handled SLA-side by
CONCRETE-TYPE INSTANTIATION: one concrete form per ordered
type the splice needs (i64, f64, Vec<u8>, bool). Covers
Compare, Less, Or.

## Files

- VM: `~/workspace/task170-work/wave1b/cmp.sla` — 363 lines, 13 functions
- VM: `~/workspace/task170-work/wave1b/MANIFEST-cmp.md` (this file)
- sha256 of `cmp.sla`:
  e38f6401bf07592996c2421c1dcf4d7b1334249e07958da843eaf9023d7a29b8
  (this manifest's own hash is not recorded inside itself.)

## Functions covered vs census scope

Census scope: Compare + Or, with the generic-instantiation
prerequisite. The oracle file also defines Less and the
private isNaN helper; the brief adds Less explicitly, so
ALL of the oracle is covered (12 public + 1 private
helper = 13 functions). No oracle function is missing.

i64 (Go signed ints; exact widening, seam (a)):
`go_cmp_compare_i64`, `go_cmp_less_i64`, `go_cmp_or_i64`.

f64 (Go float64; float32 by exact widening, seam (e)):
`go_cmp_compare_f64`, `go_cmp_less_f64`, `go_cmp_or_f64`.

Vec<u8> (Go string AND []byte; identical bytewise
ordering, delegating to bytes.Compare):
`go_cmp_compare_bytes`, `go_cmp_less_bytes`,
`go_cmp_or_bytes`.

bool (gostd extension — bool is NOT in Go's Ordered;
see Divergences): `go_cmp_compare_bool`,
`go_cmp_less_bool`, `go_cmp_or_bool`.

Private helper (1): `__go_cmp_is_nan_f64` — the oracle's
`isNaN` verbatim (`x != x`); the same test as math.sla's
`go_math_is_nan`, written out here so cmp tracks the
oracle and depends on nothing but bytes.sla.

Not in this file (out of census scope for cmp): the
`Ordered` constraint itself (a frontend type fact,
seam (a)); `cmp` has no other declarations — no
structs, no error values, no constants.

## Splice position

New packages go AFTER the existing 21-file splice order
(errors, math, strconv, strings, fmt, sync, time, sort,
runtime, io, bytes, hash, crc32, fnv, base64, hex, utf8,
path, flag, bufio, log, context): **cmp is appended at
the end, position 22**. It needs bytes.sla
(`go_bytes_compare`, position 11), which precedes it.
cmp is a dependency of slices/maps only conceptually:
slices and maps take their comparators from the
frontend, they do not call cmp functions, so cmp does
not need to precede them and nothing in the current
splice depends on cmp; append-at-end is safe.

## Cross-file dependencies

- bytes.sla — `go_bytes_compare` only (the Vec<u8>
  form). No dependency on math.sla: the f64 NaN test
  is the local `__go_cmp_is_nan_f64` (`x != x`), not
  `go_math_is_nan`, although the two are the same test.
- No other cross-file calls. No io, fmt, errors, time,
  sync, or strconv use.

## Loud rejections + panic codes claimed

NONE claimed, and no function in this file panics.

| code/id | status |
|---------|--------|
| 77433 | NOT used, stays UNCLAIMED. Assigned meaning was "f64 comparison attempted before the float surface lands — loud refusal". The float surface DOES exist in the corpus (f64 is a first-class SLA type; math.sla uses f64 comparisons throughout and flag.sla stores f64 values), so the f64 forms are implemented for real and no loud-refusal stub was needed. |
| 900044 | NOT used, stays UNCLAIMED. Assigned only if an error sentinel were defined; cmp has no error values in Go, so no sentinel exists and none was invented. |

Nothing is a silent stub. If a future wave finds an
ordered type the corpus cannot represent, that wave
claims a fresh code — it must not reuse 77433 for a
different meaning.

## Collisions found

None found. Grep-verified before writing:
- `77433`: no match in `~/workspace/task170-work/wave1a/`,
  `wave1a-sunny/`, `census/`, `drive/`,
  `drive/staging/gosla-staging/gostd/` — and no match
  anywhere under `~/workspace/task170-work/` at all.
- `900044`: same scope, same result — no match anywhere.
- The `774xx` / `9000xx` numbers already present in
  wave1a + wave1a-sunny (77401–77452 range, 900000–
  900035 range) do not include 77433 or 900044.
- No other wave-1b output directory existed when this
  file was written (`wave1b/` was created by this line),
  so no cross-package claim conflict inside wave 1b
  could be checked yet — the integration line should
  re-grep 77433/900044 across wave1b once the sibling
  packages land, to confirm they stayed unclaimed.

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **Ordered is a frontend type fact (a).** The frontend
   only emits these calls for a T satisfying Go's
   Ordered (or, for the bool form, a bool operand
   explicitly ordered false < true). Unlisted types
   (uint64, uintptr, complex, unmapped named ordered
   types) are a frontend refusal, not a silent cast —
   except the exact widenings in seams (a) and (e).
2. **Or is variadic in Go (b).** Every Or form takes a
   single Vec<T>; the frontend collects call-site
   arguments into it. The headline chaining form is
   `go_cmp_or_i64` over a Vec<i64> of Compare results.
3. **Unsigned ints (c).** No u64 instantiation: uint32
   and smaller fit i64 exactly and may map onto the
   i64 form; uint64/uintptr above MaxInt64 must be
   refused by the frontend (or a later wave adds
   `go_cmp_compare_u64`). Never silently truncate.
4. **No error values (d).** cmp returns no errors; no
   GosError surface, no target id.
5. **Float32 (e).** The f64 form serves float32 by
   exact widening (NaN / -0.0 behaviour identical
   after widening).
6. **Pure package (f).** No mutable globals, no threads,
   no closures; the block-closure parser restriction
   never arises here.

## Divergences recorded (deliberate, in-file)

- bool is NOT in Go's Ordered: `cmp.Compare[bool]`
  does not compile in Go. The bool forms (false < true)
  are a gostd extension for the slices/maps lines, NOT
  a mapping of any Go source form — the frontend must
  NEVER emit `go_cmp_compare_bool` for a Go
  `cmp.Compare[bool]` call (that source is a Go compile
  error, to be reported as one).
- `__go_cmp_is_nan_f64` duplicates `go_math_is_nan`'s
  test deliberately (oracle fidelity + keeping cmp's
  only cross-file dependency bytes.sla).
- Less for f64 is written with nested ifs rather than
  the oracle's `(isNaN(x) && !isNaN(y)) || x < y`, so
  the eager && / || semantics (task 156 defect
  (xviii)) cannot change the result.
- Or's variadic shape is a Vec<T> parameter (seam 2).

## Oracle-verification status

- Oracle file read in full: `~/workspace/task170-work/wave1b-oracle/cmp/cmp.go`
  (70 lines: Ordered, Less, Compare, isNaN, Or). Every
  oracle function is covered (see above); the Ordered
  constraint's member list was checked against the
  instantiations (signed ints -> i64, float32/64 -> f64,
  string -> Vec<u8>; uint widths and uintptr are the
  documented gap in seam 3).
- Cross-file delegate verified by reading:
  `go_bytes_compare` in
  `~/workspace/task170-work/wave1a-sunny/gostd-live-bytes.sla`
  (lexicographic bytewise, returns -1/0/+1 — the exact
  Compare contract); strings.sla has NO Compare function
  (grep-verified), so bytes is the only delegate.
- Float face verified by reading math.sla
  (`go_math_is_nan` is `x != x`; f64 `<`, `==`, `!=`
  against 0.0 literals used throughout) and flag.sla
  (f64 struct fields, `0.0` literal,
  `go_strconv_format_float`). No float-surface gap was
  found, which is why 77433 stays unclaimed.
- Splice order verified against
  `~/workspace/task170-work/splice_full.py` (the 21-file
  `order` list); cmp needs only bytes.sla from it.
- Could NOT verify: nothing about the oracle was left
  unverified — the oracle is a single 70-line file and
  was read whole. What remains unverified is the SLA
  text itself (see Verification below).

## TODOs left

- None in the .sla file. No TODO, no stub, no invented
  number. The only open items are the integration seams
  above (frontend decisions, not code TODOs) and the
  re-grep note under Collisions (integration line to
  confirm 77433/900044 stayed unclaimed once sibling
  wave-1b packages land).

## Verification

- NOT compiled, NOT tested (write-first batch, per task brief).
- sha256 of `cmp.sla` (363 lines):
  e38f6401bf07592996c2421c1dcf4d7b1334249e07958da843eaf9023d7a29b8
- Self-checks run on the written file (not compilation):
  13 `^fn ` definitions (12 public + 1 private, matching
  the manifest list); zero `panic(` calls (no loud
  refusal is reachable, consistent with no codes
  claimed); zero block-closure tokens (no `^||` text
  anywhere in the file, code or comments); the strings
  `77433` and `900044` occur only in header comments
  stating they are unclaimed.
