# MANIFEST — gostd regexp (task 170, wave 1b)

Writer line: regexp package, minimal subset. Write-first, no
compilation, no tests, no box work, no pushes.
Oracle: Go 1.27.2 stdlib `src/regexp/regexp.go` (1350 lines),
`src/regexp/syntax/syntax_regexp.go` (464),
`syntax_parse.go` (2236), `syntax_compile.go` (296),
`syntax_prog.go` (349), `syntax_simplify.go` (151),
`syntax_doc.go` (143), `exec.go` (554), `onepass.go`
(508), `backtrack.go` (365) — all read 2026-11-27.
Census: CENSUS-170.md rev1 §7.2.1, 1b table (line 272):
regexp, minimal subset first — Compile +
Match/MatchString + FindString/FindAllString +
ReplaceAllString, patterns limited to literals and
basic metacharacters (. * + ? [...] ^ $ |); full
syntax (groups, backreferences, etc.) second wave.

## Files

- VM: `~/workspace/task170-work/wave1b/regexp.sla` — 1506 lines, 44 functions
- VM: `~/workspace/task170-work/wave1b/MANIFEST-regexp.md` (this file)
- sha256 of `regexp.sla`:
  019d7190ee7427b29217dc193944acc066c4981eaa3ef9118b14c0ac6020323e

## Functions covered vs census scope

Census scope for wave 1b: Compile + Match/MatchString +
FindString/FindAllString + ReplaceAllString. ALL covered
(22 public + 22 private helpers = 44 functions).

Public (22):
`go_regexp_compile`, `go_regexp_must_compile`,
`go_regexp_err_syntax` (900038 sentinel accessor),
`go_regexp_quote_meta`, `go_regexp_string`,
`go_regexp_num_subexp` (always 0 — no groups),
`go_regexp_match_string`, `go_regexp_match`,
`go_regexp_match_string_pattern`,
`go_regexp_match_pattern` (package-level MatchString /
Match; the `_pattern` suffix avoids clashing with the
method names),
`go_regexp_find_string`, `go_regexp_find`,
`go_regexp_find_string_index`, `go_regexp_find_index`,
`go_regexp_find_all_string`, `go_regexp_find_all`,
`go_regexp_find_all_string_index`,
`go_regexp_find_all_index`,
`go_regexp_replace_all_string`,
`go_regexp_replace_all_literal_string`,
`go_regexp_replace_all`,
`go_regexp_replace_all_literal`.

Small in-scope additions beyond the census line, all
riding the same engine (no new machinery):
`go_regexp_find` / `go_regexp_find_index` /
`go_regexp_find_all` / `go_regexp_find_all_index`
([]byte forms of the named String forms),
`go_regexp_replace_all*` ([]byte forms),
`go_regexp_replace_all_literal_string` /
`go_regexp_replace_all_literal` (one boolean through
the shared replace engine), `go_regexp_quote_meta`
(pure, 1 function), `go_regexp_string`,
`go_regexp_num_subexp`, `go_regexp_err_syntax`.

Private helpers (22): `__go_regexp_append`,
`__go_regexp_slice`, `__go_regexp_append_range`,
`__go_regexp_push_code_text`,
`__go_regexp_fail`, `__go_regexp_emit`,
`__go_regexp_patch`, `__go_regexp_peek`,
`__go_regexp_peek_at`, `__go_regexp_is_name_char`,
`__go_regexp_escape`, `__go_regexp_parse_class_char`,
`__go_regexp_parse_class`, `__go_regexp_parse_atom`,
`__go_regexp_parse_concat`, `__go_regexp_parse_alt`,
`__go_regexp_class_match`, `__go_regexp_match_here`,
`__go_regexp_find_from`, `__go_regexp_expand`,
`__go_regexp_replace_all`, `__go_regexp_find_all_index`.

Structs: `GoRegexpInst` (kind/a/b),
`GoRegexpClass` (negated + lo,hi range pairs),
`GoRegexp` (expr + prog + classes),
`GoRegexpMatch` (found/start/end),
`GoRegexpParser` (pat/pos/prog/classes/err).

Compiled form: a flat instruction list
(MATCH/LIT/ANY/CLASS/BOL/EOL/SPLIT/JMP) over byte
values, interpreted by the recursive backtracking
matcher `__go_regexp_match_here` (greedy-first =
Go's leftmost-first for this subset). No NFA/one-pass
engine, per the brief.

Not in this file (out of census scope for wave 1b):
regexp/syntax (separate later package, §7.2.2 —
not written here, noted as a later dependency);
groups/submatches (FindSubmatch*, Expand public,
SubexpNames, $1 expansion); CompilePOSIX, Longest,
MatchReader/FindReader*, Split, All* iterators
(Go 1.27 iter.Seq, no SLA spelling yet).

## Loud rejections + panic codes / target ids claimed

Collision check (grep 2026-11-27, before writing):
77427, 77428, 900038, 900039 appear NOWHERE in
`wave1a/`, `wave1a-sunny/`, `census/`, `drive/`,
`staging/` (text files), and the full 774xx/9000xx
map of `wave1a/` + `wave1a-sunny/` shows no use of
them. (The `gosla-staging.tar.gz` was not grepped as
text — it is a gzip; per SPLICE-REPORT-170.md its
contents are the same staging copies already grepped.)
Collisions found: none.

| code/id | where | meaning |
|---|---|---|
| 77427 | parser: `(`, `)`, `(?...)`, `{n}`, `{n,}`, `{n,m}`, lazy `*?`/`+?`/`??`, `\d \D \w \W \s \S \b \B \A \z \x \u \U \p \P \c \Q \E`, octal `\0`-`\9`, `[[:...:]]` | pattern uses out-of-scope syntax (valid Go, second wave). Panics LOUDLY, including inside Compile — see divergence (e) in the file header. |
| 77428 | `go_regexp_must_compile` on a Compile error | Go panics with the error text; SLA panic() takes only a code. |
| 900038 | every Compile error (via `go_errors_new_with_id`); accessor `go_regexp_err_syntax()` | syntax-error sentinel; `errors.Is(err, go_regexp_err_syntax())` matches by id. Error message is Go-format: `error parsing regexp: <code>: \`<expr>\``. |
| 900039 | — | NOT claimed. Reserved for a future regexp sentinel; nothing in the minimal subset needs a second one. Left free deliberately. |

In-scope invalid patterns return a 900038 error
instead of panicking: unclosed `[` (code 1 text
"missing closing ]"), bad range `[z-a]` (2),
invalid escape (3), trailing backslash (4), `*`/`+`/`?`
with no argument (5), stacked repeats `a**` (6).
Internal codes 1–7 live only in this file (see the
`GO_REGEXP_ERR_*` consts); they are not panic codes.

No other function in this file panics. Nothing is a
silent stub.

## Splice position

New packages go AFTER the existing 21-file splice
order (errors, math, strconv, strings, fmt, sync,
time, sort, runtime, io, bytes, hash, crc32, fnv,
base64, hex, utf8, path, flag, bufio, log, context):
**regexp is 22nd, last.** Two reasons: (1) it
depends only on errors.sla, so it can sit anywhere
after errors, and (2) putting it last leaves the
verified 21-file order untouched — no re-verified
middle insertion for a new wave-1b package.

## Cross-file dependencies

- errors.sla ONLY: `go_errors_nil`,
  `go_errors_new_with_id` (Compile errors +
  sentinel). No strings/bytes/utf8 call is made:
  the parser and matcher work byte-by-byte with
  in-file helpers, so the brief's assumed
  strings/bytes/utf8 dependencies do not exist.
  (The byte-vs-rune consequence of that choice is
  seam (a) in the file header.)

## Oracle-verification status

Read, 2026-11-27 (see header for line counts):
regexp.go (full), syntax_regexp.go (full),
syntax_parse.go (Error/ErrorCode type, repeat(),
`^`/`$` parse cases), syntax_doc.go (syntax list),
exec.go (doMatch/find + matches()/replaceAll
empty-match rules), onepass.go / backtrack.go /
syntax_compile.go / syntax_prog.go /
syntax_simplify.go (skimmed for engine shape —
enough to confirm this file deliberately does NOT
reproduce them; the backtracker replaces them).

Verified against the oracle:
- Error message format `error parsing regexp:
  <code>: \`<expr>\`` (syntax_parse.go Error()).
- ErrorCode texts used (all seven, verbatim).
- `a**` is an error (repeat(), ErrInvalidRepeatOp);
  `^*` is LEGAL (repeat() rejects only pseudo-ops
  and stacked repeats) — this file accepts it.
- `^`/`$` are LINE anchors by default
  (OpBeginLine/OpEndLine, syntax_parse.go
  :954–963) — this file matches after/before `\n`.
- `.` is any-char-except-newline by default
  (OpAnyCharNotNL).
- FindAll/ReplaceAll empty-match rules (matches()
  ignores an empty match abutting the previous
  match; replaceAll skips replacement for an empty
  match abutting the previous match unless it is at
  position 0) — mirrored in
  `__go_regexp_find_all_index` /
  `__go_regexp_replace_all`.
- `$` expansion rules (Expand/extract: `$$` → `$`,
  `$0` whole match, `$1x` is one name, malformed
  `$` stays literal) — mirrored in
  `__go_regexp_expand`, with $1+ as empty (no
  submatches).

NOT verified against the oracle:
- Whether Go accepts or rejects a malformed `{`
  such as `{abc}` (this file treats a `{` not
  followed by digit/comma as a literal; the debug
  phase should confirm against syntax_parse.go).
- Any behaviour by running code: NOT compiled,
  NOT tested (write-first).

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **Byte-oriented matching (a).** `.` consumes one
   byte, literals/classes are byte ranges 0–255.
   ASCII results equal Go; multi-byte UTF-8 differs.
   Rune decoding (utf8.sla) is a second-wave matcher
   change, not a program-form change.
2. **No submatches (b).** NumSubexp is 0; `$1`+ in
   ReplaceAllString expand to empty. Submatch entry
   points arrive with groups in wave 2.
3. **Missing surface (c).** MatchReader,
   FindReader*, CompilePOSIX, Longest, Split,
   public Expand, SubexpNames, All* iterators are
   absent; the frontend must refuse them, not map
   them onto functions here.
4. **Fuel guard + recursion depth (d).** Each search
   gets (len(text)+1)*(len(prog)+2)+16 steps; the
   matcher recurses ~3 frames per input byte under
   `a*`. The debug phase must confirm the runtime
   tolerates that depth on long inputs.
5. **Out-of-scope panics inside Compile (e).** Go's
   Compile would SUCCEED on groups/`\d`/`{n,m}`;
   this file panics 77427. If the debug phase wants
   Compile to return a 900038 error instead, that is
   a one-spot change (parser panic sites → fail()),
   but it must be a decided change, not a silent one.
6. **Debug-phase dialect checks (unprobed forms).**
   Field push through a borrowed struct
   (`p.prog.push(...)` in `__go_regexp_emit`),
   Vec-handle copy out of a local struct
   (`GoRegexp { expr: p.pat, ... }` in Compile),
   struct values in Vecs (`Vec<GoRegexpInst>` etc.
   — precedented by fmt.sla's `Vec<GosVal>`),
   `while cond && !flag` loop conditions (only two
   sites, both pure operands — safe under eager
   &&), tuple returns carrying a struct
   (`(GoRegexp, ptr)`). None of these is assumed
   verified; the debug phase owns confirming them.
   Also: this file uses NO `break`/`continue`, NO
   `^||` closures, and `&&`/`||` only on pure scalar
   operands (nested ifs guard every index) — see
   task 156 defect (xviii).

## Divergences recorded (deliberate, in-file)

- ^/$ are line anchors (Go default), not text-only.
- Byte-oriented, not rune-oriented (seam 1).
- `^*` etc. accepted, as in Go.
- `{` not starting a repeat count is a literal;
  `{n}` forms panic 77427.
- FindString/Find return (value, found) so no-match
  is distinguishable from empty match (Go uses ""
  for both); FindIndex uses GoRegexpMatch with a
  found flag instead of a nil slice.
- ReplaceAll advances one byte past an empty match
  (Go: one rune).
- Panic values are codes (77427/77428); the pattern
  is not carried in the panic (it is in the Compile
  error message).
- The parser here is a minimal recursive-descent
  parser, NOT syntax.Parse; regexp/syntax is a
  separate later package and must not be assumed to
  share this instruction list.

## TODOs left in this work

- 900039 intentionally unclaimed (see table above).
- `{abc}` behaviour unverified (see oracle status).
- Lazy quantifiers, groups, `{n,m}`, `\d` etc. are
  77427 panic sites awaiting wave 2 — no code was
  invented for them.
- regexp/syntax package NOT written (separate later
  package; this file must not be mistaken for it).

## Verification

- NOT compiled, NOT tested (write-first batch, per
  task brief).
- sha256 of `regexp.sla` (VM):
  019d7190ee7427b29217dc193944acc066c4981eaa3ef9118b14c0ac6020323e
  (1506 lines, 44 functions: 22 public + 22 private;
  panic sites use only 77427/77428; only target id
  used is 900038.)

## Fix — defect (lxii) landed 2026-11-27 (Muse relay 17379)

`__go_regexp_parse_concat` (fn 15, regexp.sla:859-950) failed the
28-file diag check with `if expression branch type mismatch`: the
q==43 branch ended in a discarded i64 call
(`__go_regexp_emit(p, GO_REGEXP_INST_SPLIT, atom_start, e + 1);`)
while the else branch ended in a void if. The checker types an
if-expression's branches by each branch's last statement value, so
it saw i64 vs void. Pre-existing writer defect, unmasked by the
(lvi) fix letting the check reach regexp.sla — minimal repro fails
identically on both the c82c1c25 and a59d2579 plugins, NOT a (lvi)
regression. Registered as gosla writer-side source defect (lxii),
not a compiler regression.

Fix (1 line, regexp.sla:915): bind the return —
`let _e = __go_regexp_emit(p, GO_REGEXP_INST_SPLIT, atom_start, e + 1);`.
Before sha: bed8bfef8b703068dadda12380da456784d9b2fa7b9d37e52e0aed5821aa1fc3.
After sha: 019d7190ee7427b29217dc193944acc066c4981eaa3ef9118b14c0ac6020323e
(1506 lines, unchanged count). With this fix plus the (lxiii) fix,
the 28-file diag check passes rc=0 (first full completion).
Report: LVI-POSTLAND-CHECK-170.md §3; landing report
LXII-LXIII-LANDING-170.md.
