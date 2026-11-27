# MANIFEST — gostd maps (task 170, wave 1b)

Writer line: maps package. Write-first, no compilation, no tests.
Oracle: Go 1.27.2 stdlib `src/maps/maps.go` (112 lines) and
`src/maps/iter.go` (56 lines), both read 2026-11-27.
Census: CENSUS-170.md rev1 §7.2.1 — maps, 1b package with the
generic-instantiation prerequisite handled SLA-side by
concrete-type instantiation (see below).
Needs bytes only (go_bytes_equal / go_bytes_clone); no other
wave-1b sibling is a prerequisite.

## Files

- VM: `~/workspace/task170-work/wave1b/maps.sla` — 1160 lines, 72 functions, 8 structs
- VM: `~/workspace/task170-work/wave1b/MANIFEST-maps.md` (this file)
- sha256 of `maps.sla`:
  63ae2133cc4a8924d5867f8da07be56e59a669449efc986c585a1df24d8278a5
  (this manifest's own hash is not recorded inside itself.)

## Functions covered vs census scope

Census scope: maps, generic instantiation first, `go_maps_*`.
The Go API surface is maps.go (Equal, EqualFunc, Clone, Copy,
DeleteFunc — plus Clear, which lives in Go's builtin `clear`)
and iter.go (All, Keys, Values, Insert, Collect). Covered:

Per instantiation (4 instantiations × 18 functions = 72):
`GoMapsStringI64`, `GoMapsStringBytes`, `GoMapsStringBool`,
`GoMapsI64I64`, each with an entry struct
(`GoMapsEntryStringI64`, `GoMapsEntryStringBytes`,
`GoMapsEntryStringBool`, `GoMapsEntryI64I64`).

Instantiations covered:
- `map[string]i64` -> GoMapsStringI64
- `map[string][]byte` -> GoMapsStringBytes (also spells
  `map[string]string`: Go string and []byte are both Vec<u8>)
- `map[string]bool` -> GoMapsStringBool
- `map[i64]i64` -> GoMapsI64I64 (also spells map[int]int /
  map[int64]int64: Go int is i64 in this file's contract)

Per instantiation, maps-package API (9):
`go_maps_new_*`, `go_maps_keys_*`, `go_maps_values_*`,
`go_maps_equal_*`, `go_maps_equal_func_*`, `go_maps_clone_*`,
`go_maps_copy_*`, `go_maps_delete_func_*`, `go_maps_clear_*`.

Per instantiation, frontend language-operation surface (4,
NOT Go maps-package API — the frontend's emission surface
for m[k], m[k] = v, delete(m, k), len(m); see seam (c)):
`go_maps_get_*`, `go_maps_set_*`, `go_maps_delete_*`,
`go_maps_len_*`.

Per instantiation, iterator refusals (3, all panic 77443):
`go_maps_all_*`, `go_maps_insert_*`, `go_maps_collect_*`.

Per instantiation, private helpers (2):
`__go_maps_find_*` (index of key, or -1),
`__go_maps_remove_at_*` (rebuild entries without idx).

Not in this file: no other maps-package function exists in
Go 1.27.2. Keys/Values are provided as eager Vec snapshots,
NOT as Go's iter.Seq iterators (seam (a)); All/Insert/
Collect exist only as loud-refusal stubs.

## Splice position

New packages go AFTER the existing 21-file splice order
(errors, math, strconv, strings, fmt, sync, time, sort,
runtime, io, bytes, hash, crc32, fnv, base64, hex, utf8,
path, flag, bufio, log, context) — i.e. maps is position
22, or after slices/cmp if those wave-1b siblings land
first; the order among the three does not matter because
maps depends on none of them. It sits last because it is
a leaf: nothing earlier needs it, and it needs only bytes
(go_bytes_equal / go_bytes_clone, both in an earlier
splice file). No other cross-file calls exist in this file.

## Cross-file dependencies

- bytes.sla: `go_bytes_equal` (string-key comparison and
  []byte-value comparison in Equal) and `go_bytes_clone`
  (copying a Vec<u8> key/value out of a map without moving
  it out of the source — the move-discipline seam (e)).
  Both are public functions of an earlier splice file;
  no new cross-file surface was needed.

## Loud rejections + panic codes claimed

| code  | where | meaning |
|-------|-------|---------|
| 77443 | All / Insert / Collect stubs, all four instantiations (12 `panic(77443)` call sites) | maps.All / Insert / Collect take or return an iter.Seq/Seq2; SLA has no iterator / range-over-func surface. The stubs are the loud backstop (io.Pipe / 77402 shape); the frontend must refuse those call sites. |

No other function in this file panics. Nothing is a silent stub.

77432 NOT claimed (deliberately unclaimed, per the brief's
"only if you actually need it"): there is no SLA surface
that can detect a nil fn value, so a nil eq/del argument
cannot be checked at runtime. A nil comparator is a
frontend refusal at the call site (same class as log's
nil-writer refusal), not a runtime check.

No target ids claimed: 900042 and 900043 both left
unclaimed. Go's maps package has no error values, so this
file defines no error sentinel — as the brief anticipated,
no target ids are needed. Do not invent sentinels.

## Collisions found

grep-verified before use across `wave1a/`,
`wave1a-sunny/`, `census/` and
`drive/staging/gosla-staging/gostd/`:

- 77432: zero occurrences anywhere — free (left unclaimed, above).
- 900042 / 900043: zero occurrences anywhere — free (left unclaimed, above).
- 77431 → 77443 (RESOLVED by R1424, unanimous 10-0 =
  maps_renumber, Muse relay 16669; executed 2026-11-27,
  CONTEXT-B-FIX-170.md): the iterator stubs panicked
  77431, which sits inside sort's reserved block
  77430–77439 (sort.sla:35 reserves the block, repeated
  in bufio.sla:43 and log.sla:47 and NOTES-runtime.md:71;
  sort itself uses no codes in it and contains zero
  `panic(` calls; fmt.sla's header records that fmt was
  renumbered FROM 77430/77431 to 77446/77447 precisely
  to stay clear of the sort block, NOTES-text.md:113-115).
  Per R1424 the 12 sites were mechanically renumbered to
  77443. 77443 was verified free across the whole corpus
  (all .sla + manifests + IO-CONTRACT-170.md: zero
  occurrences before the renumber) and lies outside
  sort's block. The carve-out question is moot -- no code
  of maps' remains in 77430–77439. (Correction: the
  earlier note claiming "NO live panic(77431) exists
  anywhere" was stale -- 12 live sites existed in
  maps.sla itself; the "no live" claim held only outside
  maps.sla.) This finding is also recorded in maps.sla's
  header panic-code block.

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **Iterators (a).** All/Insert/Collect are 77443 stubs.
   Keys/Values are EAGER Vec snapshots in insertion order,
   not iter.Seq — the snapshot shape the frontend's range
   statement already rides on (runtime.sla's
   go_runtime_hashmap_keys_*). Laziness is lost; the
   key/value set is identical.
2. **Comparator / fn-value seam (b).** EqualFunc and
   DeleteFunc take fn-typed parameters (`eq: fn(V, V)
   -> bool`, `del: fn(K, V) -> bool`) — the bufio-split /
   sync-Range shape, called directly. The frontend must
   LIFT each Go func literal to a named top-level fn;
   a closure literal at such a call site is a frontend
   refusal (R1400/S3: closures can never be fn_ptr
   values). Same-instantiation EqualFunc only; cross-
   value-type EqualFunc (Go allows V1 != V2) is a
   frontend-generated loop riding on go_maps_get_*.
   Nil eq/del is a frontend refusal (77432 unclaimed).
3. **Language operations (c).** go_maps_get_* / set_* /
   delete_* / len_* are the frontend's emission surface
   for m[k], m[k] = v, delete(m, k), len(m). Get returns
   (zero, false) on a miss. Set overwrites in place.
4. **Nil maps (d).** No nil spelling: new_* returns an
   empty non-nil map; Clone of empty returns empty (Go
   Clone(nil) returns nil — divergence). The frontend
   must refuse writes to a nil map at the call site.
5. **Move discipline (e).** Vec::push moves; fn args do
   not consume. Every pushed Vec<u8> comes from
   go_bytes_clone; every helper call passes Vec<u8>
   directly as a fn argument. Debug-phase checks:
   IC-1, indexed-element field assignment through a
   borrow (`m.entries[idx].v = v` in go_maps_set_*;
   FALLBACK: remove_at + push, moving the overwritten
   entry to the end — still a valid unspecified order);
   IC-2, chained index-then-field reads as fn arguments
   (`m.entries[i].k` — the flag.sla precedent
   `__go_flag_eq(fs.defs[i].name, name)` is exactly this
   shape, so it is expected to check).
6. **HashMap alternative.** runtime.sla models Go
   map[K]V as generic HashMap<K, V> and sync.sla's
   GoSyncMap<K, V> wraps one. This file uses the
   association list the brief orders (the HashMap key-
   enumeration surface is itself unconfirmed per
   runtime.sla). A HashMap-backed rework is a NAMED
   integration decision; the public names
   (go_maps_<Fn>_<K>V) are representation-independent,
   so a rework swaps struct bodies + private helpers
   and leaves call sites untouched.

## Divergences recorded (deliberate, in-file)

- Lookup/insert/delete/Equal are LINEAR in the number of
  entries — no hashing anywhere. Cost stated in the header.
- Keys/Values are eager snapshots, not iterators (seam 1).
- Iteration order is insertion order, not Go's randomised
  order (still a valid unspecified order).
- Clone copies byte contents: Go's Clone is shallow (the
  new map shares each []byte's backing array). SLA Vec<u8>
  values are owned, so cloned keys/values are fresh copies
  via go_bytes_clone; mutating a []byte obtained from the
  clone does not affect the source's value (in Go it would).
- Clone of an empty map returns an empty map, not nil.
- DeleteFunc visits each entry exactly once, in insertion
  order — the deterministic member of Go's allowed
  behaviours (Go ranges while deleting, so an entry may or
  may not be visited).
- Equal/EqualFunc take two maps of the same instantiation;
  named Go map types share that representation. Cross-
  value-type EqualFunc is frontend-generated (seam 2).
- Go's "no special handling for non-reflexive keys (NaN)"
  note is moot: neither key type here (string, i64) has NaN.
- Equal for map[string][]byte compares values byte-wise
  (Go []byte has no == either, so this is the Go-correct
  spelling, not an approximation).

## Oracle-verification status

- Read: `wave1b-oracle/maps/maps.go` (Equal, EqualFunc,
  Clone, Copy, DeleteFunc — all five covered) and
  `wave1b-oracle/maps/iter.go` (All, Keys, Values, Insert,
  Collect — Keys/Values as snapshots, All/Insert/Collect
  as 77443 stubs). Both files read in full before writing.
- Could not verify: nothing in the oracle was unverifiable
  — but the maps.go `Clone` implementation is a
  `//go:linkname` to the runtime (`clone` in the runtime
  package), so its internals were not in the oracle; the
  shallow-clone semantics are taken from Go's documented
  Clone contract ("a shallow clone: the new keys and
  values are set using ordinary assignment") plus the
  Clone divergence note above.
- Census cross-check: CENSUS-170.md rev1 §7.2.1 table row
  for maps ("前置：泛型实例化先行，同上", `go_maps_*`)
  matches what was written.

## Verification

- NOT compiled, NOT tested (write-first batch, per task brief).
- sha256 of `maps.sla` (1163 lines, 72 functions, 8 structs):
  afa892be0abe81d87abf3ccedc5e93312b80e98fb8e8a044c36359358ec37f3c
- Static checks run by hand (grep, no compilation):
  12 `panic(77443)` call sites, all in the All/Insert/
  Collect stubs; no other 774xx or 9000xx number appears
  in code (77432 and 900042/900043 appear only in comments
  as unclaimed); no `^||` block closures, no `&&`/`||`
  (eager-evaluation defect xviii avoided by nested ifs);
  no duplicate fn or struct names; single import
  (`sa_std/vec.sa`).

## TODOs left

1. **77431 → 77443 renumber (see "Collisions found"):**
   DONE -- R1424 (unanimous 10-0 = maps_renumber) settled
   the renumber route; the 12 iterator-stub sites were
   mechanically renumbered 77431 → 77443 on 2026-11-27
   (CONTEXT-B-FIX-170.md). No carve-out is needed.
2. **IC-1 / IC-2 (see seam 5):** debug phase to confirm
   indexed-element field assignment through a borrow and
   chained index-then-field reads as fn arguments; the
   remove_at + push fallback for IC-1 is documented in
   the file header.
3. **Frontend-generated instantiations:** any map type
   outside the four covered here (map[string]any,
   map[int]string, map[string]SomeStruct, ...) needs a
   frontend-generated instantiation block copied from
   one of the four, with its own key/value comparison
   helpers in the seam-(b) shape. None is attempted here.
4. **HashMap rework decision (see seam 6):** if the
   integration phase prefers HashMap<K, V> backing, the
   rework touches only struct bodies + __go_maps_find_* /
   __go_maps_remove_at_*; public names stay.
