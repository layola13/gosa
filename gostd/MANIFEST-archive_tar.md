# MANIFEST — archive/tar.sla (task 170, wave 2, first 待写 package)

- File: `~/workspace/task170-work/wave2/archive_tar.sla`
- sha256: `05cf109e5fbe05b143a351b8b24270a073220dd6836f0871552c34834cdac058`
- Size: 127,586 B (at final check; 3,994 lines)
- Functions: 139 total — 10 public `go_archive_tar_*` + 129 `__go_tar_*` helpers
- Structs: 7 (`GosTarPaxRecord`, `GosTarSparseEntry`, `GosTarHeader`,
  `GosTarReader`, `GosTarWriter`, `GoTarNextResult`, `GoTarAllowState`)
- Constants: 71 (`GO_TAR_*`)
- Oracle: Go 1.27.2 `archive/tar` stdlib sources, mirrored at
  `~/workspace/task170-work/wave2-oracle/archive_tar/` (common.go,
  format.go, reader.go, writer.go, strconv.go, stat_unix.go,
  stat_actime1.go, stat_actime2.go — sha-verified against the box Go tree
  2026-11-28). Census row: "须 io；纯归档解析/生成", prefix
  `go_archive_tar_*` (CENSUS-170.md rev1; CENSUS-REVERSE-170.md line 52,
  first 待写 in census order).
- Status: WRITTEN + `sa check` PASS on a 29-file splice with the fresh
  603232ac plugin (see `WAVE2-ARCHIVE-TAR-170.md`). NOT pushed to GitHub
  (write-first; landing awaits Muse's dispatch).

## Go API coverage (exported surface, Go 1.27.2)

| Go item | SLA shape | Status |
|---|---|---|
| `Header` struct | `GosTarHeader` (all fields except `FileInfo`-only uses) | covered |
| `Reader` | `GosTarReader` | covered |
| `NewReader(r)` | `go_archive_tar_new_reader` | covered |
| `NewReaderStrict(r)` | `go_archive_tar_new_reader_strict` | covered |
| `(*Reader).Next()` | `go_archive_tar_reader_next` → `GoTarNextResult { hdr, err }` | covered |
| `(*Reader).Read(p)` | `go_archive_tar_reader_read` → `GoIoResult` | covered |
| `Writer` | `GosTarWriter` | covered |
| `NewWriter(w)` | `go_archive_tar_new_writer` | covered |
| `(*Writer).Flush()` | `go_archive_tar_writer_flush` | covered |
| `(*Writer).WriteHeader(hdr)` | `go_archive_tar_writer_write_header` | covered |
| `(*Writer).Write(p)` | `go_archive_tar_writer_write` | covered |
| `(*Writer).Close()` | `go_archive_tar_writer_close` | covered |
| `Format` consts (Unknown/V7/USTAR/PAX/GNU/STAR) | `GO_TAR_FORMAT_*` bitmask | covered |
| Format detection + set | `__go_tar_get_format` / `__go_tar_set_format` | covered |
| Typeflag consts (Reg/RegA/Link/Symlink/Char/Block/Dir/Fifo/XHeader/XGlobalHeader/GNULongName/GNULongLink/GNUSparse) | `GO_TAR_TYPE_*` (i64, corpus convention) | covered |
| `ErrHeader`, `ErrWriteTooLong`, `ErrFieldTooLong`, `ErrWriteAfterClose`, `ErrInsecurePath` | error constructors, target ids 900047–900051 | covered |
| PAX records (parse/format, time, keys, sorting) | `__go_tar_parse_pax_*` / `__go_tar_format_pax_*` / `__go_tar_pax_key` table | covered |
| PAX `atime`/`ctime`/`mtime`/`SCHILY.xattr.*` | via `allowedFormats` + PAXRecords merge | covered |
| Sparse files (reader-side: GNU 0x0/0x1/1x0 maps + old GNU map, holes) | `__go_tar_read_gnu_sparse_map_*`, `__go_tar_read_old_gnu_sparse_map`, `GosTarSparseEntry` | covered |
| V7 / USTAR / PAX / GNU / STAR headers | read + write paths | covered |
| `FileInfoHeader(hdr, fi)` | — (needs `io/fs`) | OMITTED, code 77444 reserved |
| `(Header).FileInfo()` | — (needs `io/fs`) | OMITTED, code 77445 reserved |
| `(*Writer).AddFS(fsys)` | — (needs `fs.FS`) | OMITTED, code 77448 reserved |

Corpus example: `go_archive_tar_demo_header_roundtrip() -> bool` —
builds a USTAR header block from a header and verifies it parses back
(format detection, checksum, name/size fields). A full writer→reader
roundtrip demo is not possible yet: `io.sla` exposes no in-memory sink
with a read-back accessor (`GosOffsetWriter` has no bytes accessor), so
the demo is a pure header-block roundtrip (documented in the file header).

## New codes claimed by this file

Panic codes (gostd block 77400–77499; verified free corpus-wide):
- `77400` — LIVE, used once: `__go_tar_set_format`, invalid format value.
  (77401 errors, 77402 io, 77403 bufio, 77404/77405/77406 bytes,
  77407 base64, 77408/77409 flag are taken; 77400 was free.)
- `77444` — RESERVED, not attached to any code: `FileInfoHeader` refusal
  (would need `io/fs`).
- `77445` — RESERVED, not attached: `(Header).FileInfo` refusal.
- `77448` — RESERVED, not attached: `(*Writer).AddFS` refusal.
  The three io/fs functions are OMITTED entirely (no stubs) — the
  slices-77419 precedent, not the maps-77443 stub precedent — because no
  callable SLA shape exists without `io/fs`. 77449/77450 stay free.
  77433 stays unclaimed (cmp reserves it); 77419 stays reserved (slices).

Error target ids (verified zero occurrences corpus-wide before claiming):
- `900047` ErrHeader ("archive/tar: header field too long" family via
  `__go_tar_header_error`)
- `900048` ErrWriteTooLong
- `900049` ErrFieldTooLong
- `900050` ErrWriteAfterClose
- `900051` ErrInsecurePath
- `900052` errMissData ("archive/tar: missed writing X bytes")
- `900053` errUnrefData ("archive/tar: unreferenced data")
- `900054` errWriteHole ("archive/tar: invalid sparse file hole")
- `900055` errSparseTooLong ("archive/tar: sparse map too long")
- `900056` headerError reason assembly (`__go_tar_header_error`)
- `900010` io.EOF and `900011` io.ErrUnexpectedEOF reused (not claimed).

## Seams (cross-file calls, all landed in the corpus)

- errors: `go_errors_new_with_id` / `go_errors_new` / `go_errors_nil` /
  `go_errors_is`
- io: `go_io_reader_read` / `go_io_writer_write` / `go_io_eof` /
  `go_io_err_unexpected_eof`
- strconv: `go_strconv_parse_uint` / `go_strconv_parse_int` /
  `go_strconv_format_int` / `go_strconv_itoa` / `go_strconv_quote`
- bytes: `go_bytes_clone` / `go_bytes_trim` / `go_bytes_index_byte` /
  `go_bytes_compare` / `go_bytes_has_prefix` / `go_bytes_has_suffix` /
  `go_bytes_split` / `go_bytes_equal` / `go_bytes_last_index_byte`
- path: `go_path_clean` / `go_path_split` / `go_path_join`
- time: `go_time_unix` (GoTime fields read directly; round-to-second is
  implemented locally — `__go_tar_time_round_sec` — because time.sla
  has no Round; the rounding is Go's nearest/half-away-from-zero, NOT
  truncate)

## Divergences from Go (also in the file header)

- D1 GoTime zero = `unix_ns == 0` (no zero-time helper in the corpus).
- D2 Writer has NO sparse support in Go 1.27.2 — WriteHeader never emits
  a GNU sparse map.
- D3 Go 1.27.2 Header has NO sparse fields — sparse is reader-side only;
  `GosTarHeader` has no sparse field.
- D4 Typeflag stored as i64 (corpus convention); on-disk byte is
  `hdr.typeflag as u8`.
- D5 Format detection reads the checksum via parseNumeric (octal-first,
  base-256 if the high bit is set).
- D6 formatString trailing-slash quirk ported faithfully.
- D7 readSpecialFile reads through the Reader's current-file Read, as
  the oracle does.
- D8 Writer.WriteHeader for GNU long name/linkname emits GNU `L`/`K`
  records; PAX emits `x` records; USTAR splits long names at `/`.
- verifyTime reason strings render the instant in PAX form because
  time.sla has no wall-clock String formatter (Go's `%v` renders e.g.
  "2026-11-28 19:00:00 +0000 UTC") — message-text only, format
  selection unaffected.
- V7/STAR headers leave `hdr.format` unset (0 = FormatUnknown) per the
  oracle — ported faithfully.
- `logical_remaining` is DERIVED (`__go_tar_logical_remaining`: sparse →
  last hole end − pos; else `tr.nb`), not a stored field.
- PAX keys are built via a byte-push table (`__go_tar_pax_key`, ids 0–20)
  because SLA has no string literals in code (only `@import`).
- PAX record sorting uses a local insertion sort over
  `go_bytes_compare` (no sort-package dependency).
- `&v[i]` borrow form (defect (lix)) is avoided throughout; sparse
  helpers use let-bound copies.

## Self-check (summary; full log in WAVE2-ARCHIVE-TAR-170.md)

- 29-file splice (28 landed + this file, builder transforms applied:
  `ptr` → `*GosError`, `@import` deduped) with the fresh 603232ac plugin:
  **rc=0 PASS**. Splice sha
  `d3ca89f3e07660b9a030dbfdd2b080f5ed7f03364f7057f8e299db9d61f78b3f`,
  32,007 lines.
- Installed plugin 0.1.8 control: rc=1, `UseAfterMove: raw` — the known
  pre-existing fmt 1009 (0.1.8 lacks the R1394 fix); fails in file 5 of
  29, before reaching tar. Expected.
- Single-file check: rc=1, parse error at the first struct-literal return
  of a foreign type (`GoIoResult`, line 2589) — the R1 assembly artefact:
  single-file check is not this corpus's natural unit. Expected.
- Environment notes (also in the report): `/content/sa_all` plugin tree
  is stale at 22a288d3 (no `sla_std/hashmap.sla`); the splice check needs
  `SLA_STD_DIR` = the 603232ac plugin clone's `sla_std`, and the fresh
  plugin via a recreated iso-home (the `/tmp` one died in the ~18:56
  reboot). The splice builder's `ptr` → `*GosError` rewrite is REQUIRED
  — without it the check fails on `go_errors_is` TypeMismatch.
