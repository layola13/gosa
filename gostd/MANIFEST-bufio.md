# MANIFEST — bufio.sla (task 170, wave 1a)

Writer line: bufio. Write-first, no compilation, no tests.
Oracle: Go 1.27.2 stdlib `/usr/local/go/src/bufio/` (bufio.go + scan.go).
Census: CENSUS-170.md rev1 §7.2.1, bufio, 1a no-prerequisite package.

## Files

- VM: `~/workspace/task170-work/wave1a/bufio.sla` — 1670 lines
- VM: `~/workspace/task170-work/wave1a/MANIFEST-bufio.md` (this file)
- Box: `/content/xiaohuoche-work/task170/gostd/bufio.sla` (+ this manifest)
- sha256 (VM): `e05496fa7e1ae735183908fd7952a5b2de0ac6e9138e6d8c5494ced75c4ac6a4`
- Box sha256 verified to match after scp (see §7): bufio.sla
  `e05496fa7e1ae735183908fd7952a5b2de0ac6e9138e6d8c5494ced75c4ac6a4`
  on both ends.

## 1. Functions covered vs census scope

Census scope was: Reader (NewReader/NewReaderSize, Read/ReadByte/
ReadBytes/ReadString/ReadSlice/ReadLine/Peek/UnreadByte/UnreadRune/
Discard/Buffered/Reset/Size), Writer (NewWriter/NewWriterSize,
Write/WriteByte/WriteString/WriteRune/Available/Buffered/Flush/
Reset), Scanner (NewScanner, Scan/Text/Bytes/Err/Split/Buffer, split
funcs ScanLines/ScanWords/ScanRunes/ScanBytes). All covered, plus the
extras marked (+) below, which Go's package also has and which are
needed for a complete Reader/Writer (UnreadRune without ReadRune
would be meaningless).

### Reader — 17 public functions (all census items covered)

| Go | gostd |
|---|---|
| NewReaderSize | go_bufio_new_reader_size |
| NewReader | go_bufio_new_reader |
| (*Reader).Size | go_bufio_reader_size |
| (*Reader).Buffered | go_bufio_reader_buffered |
| (*Reader).Reset | go_bufio_reader_reset |
| (*Reader).Peek | go_bufio_reader_peek |
| (*Reader).Discard | go_bufio_reader_discard |
| (*Reader).Read | go_bufio_reader_read |
| (*Reader).ReadByte | go_bufio_reader_read_byte |
| (*Reader).UnreadByte | go_bufio_reader_unread_byte |
| (*Reader).ReadRune (+) | go_bufio_reader_read_rune |
| (*Reader).UnreadRune | go_bufio_reader_unread_rune |
| (*Reader).ReadSlice | go_bufio_reader_read_slice |
| (*Reader).ReadLine | go_bufio_reader_read_line |
| (*Reader).ReadBytes | go_bufio_reader_read_bytes |
| (*Reader).ReadString | go_bufio_reader_read_string |
| (*Reader).WriteTo (+) | go_bufio_reader_write_to |

### Writer — 13 public functions (all census items covered)

| Go | gostd |
|---|---|
| NewWriterSize | go_bufio_new_writer_size |
| NewWriter | go_bufio_new_writer |
| (*Writer).Size (+) | go_bufio_writer_size |
| (*Writer).Available | go_bufio_writer_available |
| (*Writer).Buffered | go_bufio_writer_buffered |
| (*Writer).AvailableBuffer (+) | go_bufio_writer_available_buffer |
| (*Writer).Reset | go_bufio_writer_reset |
| (*Writer).Flush | go_bufio_writer_flush |
| (*Writer).Write | go_bufio_writer_write |
| (*Writer).WriteString | go_bufio_writer_write_string |
| (*Writer).WriteByte | go_bufio_writer_write_byte |
| (*Writer).WriteRune | go_bufio_writer_write_rune |
| (*Writer).ReadFrom (+) | go_bufio_writer_read_from |

### ReadWriter — 1 public function (+)

| Go | gostd |
|---|---|
| NewReadWriter | go_bufio_new_read_writer |

### Scanner — 7 public functions (all census items covered)

| Go | gostd |
|---|---|
| NewScanner | go_bufio_new_scanner |
| (*Scanner).Scan | go_bufio_scanner_scan |
| (*Scanner).Err | go_bufio_scanner_err |
| (*Scanner).Bytes | go_bufio_scanner_bytes |
| (*Scanner).Text | go_bufio_scanner_text |
| (*Scanner).Buffer | go_bufio_scanner_buffer |
| (*Scanner).Split | go_bufio_scanner_split |

### Split functions — 4 public functions (all covered)

go_bufio_scan_bytes, go_bufio_scan_runes, go_bufio_scan_lines,
go_bufio_scan_words.

### Sentinel accessors — 10 public functions

go_bufio_eof (io.EOF, id 900010 — shared, see §3),
go_bufio_err_buffer_full, go_bufio_err_negative_count,
go_bufio_err_invalid_unread_byte, go_bufio_err_invalid_unread_rune,
go_bufio_err_too_long, go_bufio_err_negative_advance,
go_bufio_err_advance_too_far, go_bufio_err_bad_read_count,
go_bufio_err_final_token.

### Result structs (all public, Go multi-returns as structs)

GoIoResult (read/write results, unified per R1400 —
GoBufioReadResult / GoBufioWriteResult deleted),
GoBufioCountResult,
GoBufioPeekResult, GoBufioByteResult, GoBufioRuneResult,
GoBufioSliceResult, GoBufioBytesResult, GoBufioLineResult,
GoBufioWriteRuneResult, GoBufioFragments (internal),
GoBufioSplitResult. Data structs: GoBufioReader, GoBufioWriter,
GoBufioReadWriter, GoBufioScanner.

Total: 52 public `go_bufio_*` functions + 23 private
`__go_bufio_*` helpers = 75 functions.

## 2. Loud rejections (panics — nothing silently stubbed)

| Code | Where | Go behaviour matched |
|---|---|---|
| 77415 | underlying read/write handle returns a negative count, returns n larger than the bytes it appended, or an impossible count (fill / Read / writeBuf / Flush / Write / ReadFrom / Scanner read loop) | Go panics errNegativeRead / errNegativeWrite / Scanner ErrBadReadCount path |
| 77415 | fill called on a full buffer | Go panics "bufio: tried to fill full buffer" |
| 77415 | Scanner: >100 empty tokens without progressing | Go panics "bufio.Scan: too many empty tokens without progressing" |
| 77415 | ReadLine: rewind past start of buffer (Go: unreachable) | Go panics "bufio: tried to rewind past start of buffer" |
| 77403 | Scanner.Buffer / Scanner.Split called after Scan started | Go panics "Buffer called after Scan" / "Split called after Scan" |

Value errors (returned, not panics) use GosError sentinels with
target ids — see §3. `Peek(-1)` / `Discard(-1)` return
ErrNegativeCount; `Peek` past the buffer returns ErrBufferFull;
`ReadSlice` past a full buffer returns ErrBufferFull; invalid
`UnreadByte`/`UnreadRune` return ErrInvalidUnreadByte /
ErrInvalidUnreadRune; Scanner split errors return
ErrNegativeAdvance / ErrAdvanceTooFar / ErrTooLong /
ErrBadReadCount via `Scanner.Err()`.

### Correction to the brief (verified against the oracle)

The brief suggested 77403 for "negative size where Go panics".
Go 1.27.2 does NOT panic there: `NewReaderSize` clamps with
`max(size, 16)` and `NewWriterSize` defaults `size <= 0` to 4096.
gostd does the same (clamp / default) and claims no panic for it.
77403 instead covers the genuine Go panics above (Buffer/Split
after Scan). Recorded in the file header.

## 3. Panic codes claimed

- 77415 — bufio read error / invalid state (see §2).
- 77403 — bufio buffer misuse (see §2). R1400: formally assigned
  to bufio; io.sla:12 "reserved" rewritten.
- No collision: 77401 errors, 77410/77413 sync, 77411/77412
  strconv, 77416/77417 strings, 77420/77421/77423 time,
  77430–77439 sort, 77440–77442 runtime, 77446/77447 fmt,
  77451/77452 math — all untouched (whole-library map in
  NOTES-runtime.md).

## 4. Target ids used (gostd owns 900000–999999)

- 900010 — io.EOF, SHARED. `go_bufio_eof()` builds it; bufio
  defines NO second EOF sentinel. The io line's EOF
  constructor, if it adds one, must use this same id.
  (strings.sla's `go_strings_eof()` already builds the
  identical id; matching is by id.)
- 900033 — bufio.ErrBufferFull
- 900034 — bufio.ErrNegativeCount
- 900035 — bufio.ErrInvalidUnreadByte
- 900023 — bufio.ErrInvalidUnreadRune
- 900024 — bufio.ErrTooLong (Scanner)
- 900025 — bufio.ErrNegativeAdvance (Scanner)
- 900026 — bufio.ErrAdvanceTooFar (Scanner)
- 900027 — bufio.ErrBadReadCount (Scanner)
- 900028 — bufio.ErrFinalToken (Scanner; consumed by Scan,
  never surfaced as a Scanner error)
- No collision with 900001/900002 (strconv) or 900010 (io.EOF).

## 5. Open integration seams (esp. the io seam)

1. **The io seam — DONE per R1400 (bufio_ctor_handles).**
   The underlying reader/writer is an io HANDLE retained
   in the struct (GosIoReader / GosIoWriter, defined in
   io.sla; the closure-adapter route S3 was proven
   impossible, RULING-EXEC-170.md §3.2):
   - read handle: `GosIoReader`, read through
     `go_io_reader_read(&h, dst, max) -> GoIoResult` —
     appends up to `max` bytes to the supplied Vec and
     returns `{ n, err }` (n = bytes appended; same
     convention as strings.Reader.Read, because SLA
     Vecs are handles).
   - write handle: `GosIoWriter`, written through
     `go_io_writer_write(&h, p) -> GoIoResult`.
   Constructors take handles BY VALUE (bufio owns the
   copy; do not reuse the original afterwards);
   WriteTo's writer / ReadFrom's reader are BORROWED
   (`&`). Read/write results are unified on
   `GoIoResult` (GoBufioReadResult / GoBufioWriteResult
   deleted). A reader from another package reaches
   bufio by materialising its bytes and wrapping them
   in a LimitReader handle.
   a. The named fn `go_bufio_scan_lines` is still used
      as the default split value (split fns are NOT io
      handles; untouched by this change).
   b. A handle MUST append at most `max` bytes and
      report n == bytes appended; a handle that lies
      about n is a broken reader and gostd panics 77415.
   c. io.ErrNoProgress and io.ErrShortWrite carry the
      io line's target ids (900012 / 900014, R1378):
      `__go_bufio_no_progress` / `__go_bufio_short_write`
      build them with `go_errors_new_with_id`, so
      `errors.Is` matches io's `go_io_err_no_progress` /
      `go_io_err_short_write`. (Message bytes still differ:
      bufio writes "io: no progress" / "io: short write",
      io writes the Go messages; same id, different text.)
   d. No WriterTo / ReaderFrom / StringWriter fast paths:
      Reader.WriteTo and Writer.ReadFrom / WriteString run
      the generic loop only (fast paths need interface
      assertions the handle seam cannot express). The
      generic path is always correct; it only costs copies.
2. **Splice position.** bufio sits after io in library order
   and must splice after errors + strconv (it calls
   `go_errors_*` and `__go_utf8_decode` /
   `__go_utf8_append`). Cross-file calls are plain calls.
3. **ReadWriter is by value, not shared.** Go's ReadWriter
   shares the very same Reader/Writer objects. gostd stores
   both by value: `NewReadWriter` takes ownership; do NOT
   keep using the original Reader/Writer afterwards (their
   buffered state would split from the pair's). Use the
   pair's `.r` / `.w` fields.
4. **Zero values are NOT usable** (Go's are, after Reset).
   Constructors are required; Reset on a struct whose
   buffer was never allocated grows a default one, but the
   handle fields still need a constructor or Reset call.
5. **NewReaderSize / NewWriterSize always wrap.** Go returns
   the underlying *Reader/*Writer unchanged when it is
   already large enough; the handle seam cannot test
   that, so gostd always wraps (transparent: same bytes,
   same errors).

## 6. Documented divergences from Go (all in-file, none silent)

- Peek returns a COPY of the buffered bytes, not a slice
  into the buffer (Go's bytes die at the next read;
  gostd's copy stays valid; contents at call time equal).
- Split functions receive a COPY of the unprocessed data,
  never the live buffer; Scanner.Bytes returns the stored
  token Vec (an alias, as in Go — overwritten by next Scan).
- ReadLine's nil line is an empty Vec<u8> (no nil slices
  in gostd; len 0 is the test). A non-empty final line
  without a newline returns nil error, as in Go.
- Scanner.Buffer copies the caller's buffer at full length
  (Vec exposes no capacity; Go uses buf[0:cap(buf)]).
- Writer.AvailableBuffer returns an empty Vec (no capacity
  to report); append to it and pass it to the next Write.
- A 0-byte nil-error write from the underlying writer
  spins forever in Go's Write loop too; gostd matches Go
  (hangs) rather than inventing an error.
- WriteRune encodes out-of-range / surrogate runes as
  U+FFFD via `__go_utf8_append`, matching Go's
  utf8.EncodeRune.

## 7. Delivery / verification

- Written on this VM first, then copied to the box with scp
  (SSH: `-o IdentitiesOnly=yes -o StrictHostKeyChecking=yes
  -o UserKnownHostsFile=/home/hatch/.ssh/known_hosts -i
  /home/hatch/.ssh/id_ed25519 -p 25024 root@31.58.87.230`).
- sha256 checked on both ends (match confirmed):
  - bufio.sla, VM and box:
    `e05496fa7e1ae735183908fd7952a5b2de0ac6e9138e6d8c5494ced75c4ac6a4`
  - MANIFEST-bufio.md was re-copied after this §7 fill-in and
    its sha re-verified to match on both ends (final hash in
    the delivery report).
- NOT compiled, NOT tested, per the write-first rule. No
  sa/sla invocation, no tests, no Kaggle, no push to any
  remote, no relay message, no touch of Sunny's files or
  the already-written 8 packages or runtime.sla.

### R1400 addendum (2026-11-27, bufio_ctor_handles)

- bufio.sla rewritten to the handle seam (R1400): constructors
  take GosIoReader / GosIoWriter handles, results unified on
  GoIoResult. New sha (VM and box):
  `587789ba913dc17eab0826eef1e06661590dbd61ec3ad3e9e70c555ccd824384`
  (1671 lines; was `cc015991…`, 1673 lines).
- io.sla gained the handle section (GosIoReader / GosIoWriter,
  7 constructors + 2 dispatches): new sha `b1589005…`
  (732 lines; was `c2917536…`, 563 lines). base64.sla /
  hex.sla gained the 77414-shared-intentionally note only.
- Verified on box 6e9b724f1f4b: io-handle probe RC=0; full
  6-file probe stops only at the known flush `err`
  UseAfterMove (R1394 scope, NOT fixed); scratch-diagnostic
  probe with the two known bufio 1009s patched RC=0; full
  splice 15,771 lines sha `8f7344ba…`, sa check stops at
  the known fmt `raw` UseAfterMove (log sha `46bfe001…`,
  byte-identical to pre-R1400); demos gate 28/28 GREEN.
- Full report: R1400-EXEC-170.md (local
  ~/workspace/task170-work/).

## 8. Still missing / for the integration phase

- The io seam in §5 — DONE per R1400 (bufio_ctor_handles):
  constructors take GosIoReader / GosIoWriter handles,
  results unified on GoIoResult. ErrNoProgress /
  ErrShortWrite target ids were already fixed by R1378.
- Frontend mapping (`go_bufio_*` call emission) + demos —
  not this line's scope (stdlib mapping line / Sunny).
- Debug phase (compile + run) — separate, later step run
  by others; this file is write-first only.

## 9. SLA-FIXES-170 (2026-11-27, Muse ruling 16553③) — flush + scratch fixed

- Flush `err` (bufio.sla:1156–1158): `w.err = err; return err;`
  was a real UseAfterMove once the splice's ptr→*GosError
  rewrite is accounted for (raw `ptr` is Copy, `*GosError`
  is a move type). Fix: `return w.err;` (return the stored
  field, same pointer value). Regression probes B1c (fails:
  `UseAfterMove: identifier 'err'`) / B1d (passes, rc=0).
- Writer ReadFrom `scratch` scope (bufio.sla:1303/:1319):
  `let scratch` was declared inside the inner retry loop but
  read after it (`res.n > len(scratch)`). Fix: hoist
  `let scratch = Vec::new();` before the inner loop and
  re-create it fresh each iteration (`scratch = Vec::new();`
  at loop top, preserving per-iteration-fresh semantics).
  Regression probe B2 (fails before: `UndefinedVariable:
  identifier 'scratch' is not defined in this scope`).
- New sha (VM and box):
  `7ac5af2adf47086592151b4f0fc6871cf9e43687ec0522005e577ec1796fc729`
  (1674 lines; was `587789ba…`, 1671 lines).
- Mini-splice check (errors+math+strconv+io+utf8+bufio,
  full splice's patches + rewrite) rc=0 after the fixes
  (mini sha `226a1647…`, 5278 lines; before: rc=1 flush
  UseAfterMove). Full splice 15,774 lines sha `cdbe090e…`;
  sa check still stops at `Undefined call: go_sort_vec`
  (checker defect, item 3 of SLA-FIXES-170).
- RUNTIME CAVEAT (not a check-level problem): the hoisted
  ReadFrom still cannot land data at runtime — a Vec passed
  by value to a mutating callee is cloned for the caller
  (probes P0/P2/P3 panic 44202/44401/44411; the working
  shape is `&Vec<u8>`, probe P1 rc=0). The fixed ReadFrom
  is PREDICTED to panic 77415 at the `res.n > len(scratch)`
  guard on the first successful read (P2 is the direct
  evidence; the real ReadFrom could not be run end-to-end
  because build-exe of any module containing `while true`
  no-break functions traps 1014 — see SLA-FIXES-170.md §5).
  The `dst: &Vec<u8>` API change across io.sla + bufio.sla
  needs a Muse ruling before any library edit.
- Full report: SLA-FIXES-170.md (local
  ~/workspace/task170-work/).

## 10. CONTEXT-B-FIX-170 (2026-11-27, Muse relay 16768, R1426 lib_borrow) — dst: &Vec<u8>

- R1426 (liii) reader-dst fork, unanimous 10-0 = lib_borrow: every fn
  that APPENDS into a dst takes `dst: &Vec<u8>` (io.sla's readers
  likewise). A by-value dst is a copy: pushes into it are invisible
  to the caller (the (liii) defect, SLA-FIXES-170.md §4 — the old
  ReadFrom panicked 77415 on the first successful read because
  `len(scratch)` never grew). This executes the ruling §9's RUNTIME
  CAVEAT anticipated.
- Changed in this file (2 signatures + 5 call sites):
  `__go_bufio_append` and `go_bufio_reader_read` take
  `dst: &Vec<u8>`; `__go_bufio_fill`, `go_bufio_reader_read`
  (scratch), `go_bufio_writer_read_from`, scanner-fill pass
  `&scratch`; `go_bufio_reader_read_bytes` passes `&out`.
  `dst.push(...)` / `len(dst)` / `dst[i]` bodies unchanged (push /
  index / len through a borrow). No other Vec<u8> param changed:
  `p` / `s` / `buf` / `data` params are read-only sources or
  stored, not dst.
- New sha (VM and box):
  `1daefc65e38502ae45d687529c9847636ae2fa4496ae1705506fad78f2c23bcb`
  (1684 lines; was `7ac5af2a…`, 1674 lines). io.sla (no manifest
  of its own; R1426 header note added):
  `566d5901798a33f6032d6256f693ccbe381b9309f38f36211948fcc8e37db77d`
  (742 lines; was `b1589005…`, 732 lines); 11 dst signatures +
  `__go_io_append(&w.sinks[i], p)` (borrow-of-indexed-element
  probe dstA checks rc=0).
- Check-level verification: 28-file diagnostic splice check
  reaches the slices cmp defect (CONTEXT-B-FIX-170.md §5);
  24-file mini-splice (22 + cmp + maps, Chan sites neutralised
  in scratch only) rc=0. Gate 28/28 GREEN (19 PASS + 9 REFUSED-OK).
- RUNTIME CAVEAT (new, blocks ReadFrom runtime delivery): the
  `&Vec<u8>` change checks, but pushing through a borrowed Vec
  segfaults or lands wrong values at runtime in every multi-param
  shape probed (only the single-param P1 shape runs clean; see
  CONTEXT-B-FIX-170.md §5 probe matrix, dstB2–E2, fixed plugin
  and pristine baseline plugin identical). Pre-existing emitter
  defect, NOT a 1009-fix regression. ReadFrom runtime delivery
  awaits a Muse-numbered compiler line for it.
- Full report: CONTEXT-B-FIX-170.md (local ~/workspace/task170-work/).
