# MANIFEST — gostd log (task 170, wave 1a)

Writer line: log package. Write-first, no compilation, no tests.
Oracle: Go 1.27.2 stdlib `src/log/log.go` (483 lines), read 2026-11-26.
Census: CENSUS-170.md rev1 §7.2.1 — log, 1a no-prerequisite package.
Needs io (Sunny writing io in parallel); this file writes against the
io contract in its header (Writer = `&GosBuilder`, fmt precedent) and
does not wait for her file.

## Files

- VM: `~/workspace/task170-work/wave1a/log.sla` — 771 lines, 43 functions
- VM: `~/workspace/task170-work/wave1a/MANIFEST-log.md` (this file)
- Box: `/content/xiaohuoche-work/task170/gostd/log.sla`
- Box: `/content/xiaohuoche-work/task170/gostd/MANIFEST-log.md`
- sha256 of `log.sla` (VM, verified identical on box — see below):
  b74621c6455dd450bd9fd26dd2066e7be37d02459f1331ac9592c7bed22cd01d
  (this manifest's own hash is not recorded inside itself; it was
  verified separately by the sha256sum comparison at copy time.)

## Functions covered vs census scope

Census scope: Logger struct (New, Print/Printf/Println,
Fatal/Fatalf/Fatalln, Panic/Panicf/Panicln, Output,
SetOutput/SetPrefix/SetFlags/Prefix/Flags, Writer), package-level
Print/Printf/Println/Fatal*/Panic*/SetOutput/SetPrefix/SetFlags/Output,
flags Ldate/Ltime/Lmicroseconds/Llongfile/Lshortfile/LUTC/Lmsgprefix/
LstdFlags. ALL covered (36 public + 7 private helpers = 43 functions).

Flag constants: `GO_LOG_LDATE` (1), `GO_LOG_LTIME` (2),
`GO_LOG_LMICROSECONDS` (4), `GO_LOG_LLONGFILE` (8),
`GO_LOG_LSHORTFILE` (16), `GO_LOG_LUTC` (32), `GO_LOG_LMSGPREFIX` (64),
`GO_LOG_LSTD_FLAGS` (3).

Structs: `GoLogger` (mu/out/to_stderr/prefix/flag/is_discard),
`GoLogWriterResult` (w + to_stderr bit — see Writer note below).

Constructors (3):
`go_log_new(out, prefix, flag)`, `go_log_new_discard(prefix, flag)`,
`go_log_new_std()` (the std logger's shape: stderr, "", LstdFlags).

Logger methods (17): `go_log_logger_set_output`,
`go_log_logger_set_output_discard`, `go_log_logger_flags`,
`go_log_logger_set_flags`, `go_log_logger_prefix`,
`go_log_logger_set_prefix`, `go_log_logger_writer`,
`go_log_logger_output`, `go_log_logger_print`,
`go_log_logger_printf`, `go_log_logger_println`,
`go_log_logger_fatal`, `go_log_logger_fatalf`,
`go_log_logger_fatalln`, `go_log_logger_panic`,
`go_log_logger_panicf`, `go_log_logger_panicln`.

Package-level (16, each takes the frontend-held `std: &GoLogger`
first — see the std-ownership seam below): `go_log_print`,
`go_log_printf`, `go_log_println`, `go_log_fatal`,
`go_log_fatalf`, `go_log_fatalln`, `go_log_panic`,
`go_log_panicf`, `go_log_panicln`, `go_log_set_output`,
`go_log_flags`, `go_log_set_flags`, `go_log_prefix`,
`go_log_set_prefix`, `go_log_writer`, `go_log_output`.

Private helpers (7): `__go_log_append`, `__go_log_itoa_fixed`
(Go log.itoa, zero-padded), `__go_log_parse_caller`
("file:line" split at last ':'), `__go_log_short_file`
(basename, Go's exact loop), `__go_log_format_header`
(Go formatHeader order + separators, byte-exact),
`__go_log_emit` (discard fast path, one prefix/flag load,
timestamp, caller hook, newline rule, single write),
`__go_log_write` (mutex + builder/stderr dispatch).

Not in this file (out of census scope for log): `log/slog`
(structured logging, separate package, §7.2.2) and `log/syslog`
(base gap, §7.3.1 — frontend loud refusal, not here).

## Loud rejections + panic codes claimed

gostd block 77400–77499; no collision with the whole-library map in
NOTES-runtime.md (77401 errors, 77410–77413 sync, 77416/77417 strings,
77420/77421 + 77423 time, 77430–77439 sort reserved, 77440–77442
runtime, 77446/77447 fmt, 77451/77452 math).

| code  | where | meaning |
|-------|-------|---------|
| 77422 | Fatal/Fatalf/Fatalln (logger + package-level) | Go calls os.Exit(1); no exit surface exists in gostd (os is a base gap). Fatal formats + writes the line, then panics. Process still stops, but with a panic code, not exit status 1. |
| 77424 | Panic/Panicf/Panicln (logger + package-level) | Go panics with the message string as the panic value; SLA panic() takes only a code. The message is written to the log first, then the code panics. The message string is NOT recoverable via recover(). |

No other function in this file panics. Nothing is a silent stub.

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **stderr hook (a).** The std logger writes through the
   frontend-emitted hook `fn go_host_write_stderr(b: Vec<u8>) -> void`
   (mirrors fmt's `go_host_write_stdout`). Called by log.sla, must be
   DEFINED by the frontend once per program. It returns void, so a
   failing stderr write cannot be reported: `Output` always returns a
   nil error. If the host needs write-error reporting, the hook
   signature must change to return a GosError ptr.
2. **Fatal exit seam (b).** Fatal = format + write, then panic(77422).
   Named integration decision: replace the panic with a real exit call
   if an exit surface ever lands; write-before-exit ordering must stay.
3. **Caller hook (d).** The frontend must emit
   `fn go_log_caller() -> Vec<u8>` returning `"file:line"` (full path;
   Lshortfile strips to basename here). Empty return = the file part
   is OMITTED from the header (brief's contract, differs from Go's
   `"???:0"`). The hook takes NO calldepth, so `Output`'s calldepth
   argument is accepted and IGNORED — the mapping line must arrange
   for Output call sites to report the Output caller's caller.
4. **Std-logger ownership.** SLA has no mutable globals (verified: no
   top-level static/let in sa_std or the gostd corpus), so the
   package-level `std` logger cannot live in this file. The frontend
   creates exactly ONE std logger per program with `go_log_new_std()`,
   holds it, and passes `&std` to every package-level `go_log_*` call.
   `log.Default()` maps to that held value. There is deliberately NO
   `go_log_default()` accessor (a fresh-each-call constructor under
   that name would silently hand out a different logger).
5. **Date/time are UTC, not local.** Go prints LOCAL time unless LUTC
   is set; gostd's only field getters are time.sla's, documented as
   UTC. Non-LUTC headers print UTC and LUTC is accepted but changes
   nothing. Needs a local-time extern to fix.
6. **Discard / nil writer.** `io.Discard` is modelled by
   `go_log_new_discard` / `go_log_logger_set_output_discard` (the
   mapping line calls these for a Discard source form). A nil
   `io.Writer` argument has no gostd spelling — the mapping line must
   refuse it (frontend refusal), not pass a dummy.
7. **Writer() shape.** `go_log_writer` / `go_log_logger_writer`
   return `GoLogWriterResult { w, to_stderr }`, not a bare builder:
   the stderr shape has no builder to return, and a silent empty
   builder would be a wrong value.
8. **Cross-file struct-field read.** `go_log_new` /
   `go_log_logger_writer` build a `GosBuilder` value from a
   `&GosBuilder` via `GosBuilder { buf: out.buf }` (field access
   through a borrow, same shape sync.sla uses). If the debug phase
   wants a strings.sla accessor instead, the fix is one helper there;
   semantics (shared buffer handle) do not change.
9. **Logger discipline (same as Go).** Zero values are NOT usable
   (the GoMutex token channel must exist — use the constructors);
   a Logger must not be copied after first use (copies alias the
   same token channel and output buffer).

## Divergences recorded (deliberate, in-file)

- Header date/time UTC, not local (seam 5 above); LUTC no-op.
- `Output` error always nil (builder cannot fail; stderr hook is void).
- Panic value not carried (code 77424; message preserved in the log line).
- Empty caller hook result omits the file part (Go would print `"???:0"`).
- No `log.Default()` accessor (seam 4 above).
- Header byte format otherwise follows Go's `formatHeader` exactly
  (order, separators, zero-padding, `": "` after file:line, trailing
  newline iff the line does not already end in one).

## Verification

- NOT compiled, NOT tested (write-first batch, per task brief).
- sha256 verified VM vs box after scp (2026-11-26 ~19:5x CST):
  log.sla b74621c6455dd450bd9fd26dd2066e7be37d02459f1331ac9592c7bed22cd01d
  on both ends (771 lines); MANIFEST-log.md also compared and matched
  at copy time (a1b02745… before the hash line above was added; the
  current manifest was re-copied after that edit and re-verified).
