# gosla Rust-route re-verification (2026-11-25, box sla aafd7de + tonight's tree)

Increments 1+2 (18 EMIT-OK demos) rebuilt end-to-end with the Rust
`sla build-exe` (SAB route) and diffed against expected.stdout:
**14/18 byte-identical.** The 4 failures are all upstream Rust-route
gaps, none are gosla emitter bugs (all 18 pass on the Zig route):

- 07_switch: `UnsupportedSabDirectFeature: SwitchExpr` — the SAB
  direct lowering has no switch-expression surface yet.
- 08_funcs: `error[MemoryLeak]` — pinned by minims m1-m7 to defect
  (xxxiii): a `-> void` function that reads its parameter leaks a
  register at function exit on the SAB route (CLAIMS.md).
- 15_map: `TypeMismatch: remove receiver is not a Vec, actual
  HashMap` — the Rust checker resolves `m.remove(&k)` against the
  Vec surface instead of the HashMap one (Zig route accepts).
- 16_string: `UnsupportedSabDirectFeature: string literal let
  binding` — defect (xxv) itself (string bindings blocked on the
  Rust route); gosla's part stays as-is.

Minims: box /content/sa_all/sapy/gosla-verify/m{1..7}.sla
(m1/m2 value-fn OK; m3/m4/m7 void+param-read FAIL; m5 void
param-unread OK; m6 void no-param OK).
