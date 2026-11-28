# MANIFEST — archive/zip.sla (task 170, wave 2, package 2)

Census: archive/zip, second 待写 in census order (CENSUS-REVERSE-170.md
line 53; archive/tar was line 52). Prefix `go_archive_zip_*` / `__go_zip_*`.

## File

- `wave2/archive_zip.sla` — 2,393 lines, 76,220 bytes
- sha256: `57a16e6c7f4f611c53216e68f1543bd4f2b366421af0ce01025d965b1cae762b`
- Splice position: file 30 (28 landed corpus files + archive/tar + this file)
- Splice: `wave2/splice30_zip.sla` on box 25027 — 34,399 lines,
  sha256 `eec0c626329e75dea40ba58cae1650497d9e63f48209bffa9e9ba38e2002edfd`
  (base splice29_tar.sla sha `d3ca89f3e076…`, 32,007 lines)

## Oracle (Go 1.27.2, mirrored from the box Go tree)

`wave2-oracle/archive_zip/` — all four shas match the box exactly:

| file | lines | sha256 (first 12) |
|---|---|---|
| reader.go | 988 | d35e8e55… |
| writer.go | 740 | a76c310e… |
| struct.go | 418 | b2ee362b… |
| register.go | 147 | 79c2b8c3… |

## Public API (36 functions)

Errors (Go's exported vars as constructors):

- `go_archive_zip_err_format() -> ptr` (900057)
- `go_archive_zip_err_algorithm() -> ptr` (900058)
- `go_archive_zip_err_checksum() -> ptr` (900059)
- `go_archive_zip_err_insecure_path() -> ptr` (900060)

FileHeader:

- `go_archive_zip_file_header_new() -> GosZipFileHeader`
- `go_archive_zip_file_header_mod_time(h) -> GoTime`
- `go_archive_zip_file_header_set_mod_time(h, t) -> GosZipFileHeader`
- `go_archive_zip_file_header_is_dir(h) -> bool`

Reader:

- `go_archive_zip_new_reader(data: Vec<u8>) -> GoZipReaderResult`
- `go_archive_zip_new_reader_strict(data: Vec<u8>) -> GoZipReaderResult`
- `go_archive_zip_reader_comment(r) -> Vec<u8>`
- `go_archive_zip_reader_file_count(r) -> i64`
- `go_archive_zip_reader_get(r, i) -> GosZipFile`
- `go_archive_zip_reader_base_offset(r) -> i64`

File:

- `go_archive_zip_file_header(f) -> GosZipFileHeader`
- `go_archive_zip_file_name(f) -> Vec<u8>`
- `go_archive_zip_file_is_dir(f) -> bool`
- `go_archive_zip_file_data_offset(r, f) -> GoZipHeaderResult`
- `go_archive_zip_file_open(r, f) -> GoZipFileReaderResult`
- `go_archive_zip_file_open_raw(r, f) -> GoZipRawReaderResult`
- `go_archive_zip_file_reader_read(fr, dst, max) -> GoIoResult`
- `go_archive_zip_file_reader_close(fr) -> ptr`
- `go_archive_zip_raw_reader_read(rr, dst, max) -> GoIoResult`
- `go_archive_zip_raw_reader_close(rr) -> ptr`

Writer:

- `go_archive_zip_new_writer(w: GosIoWriter) -> GosZipWriter`
- `go_archive_zip_writer_create(w, name) -> ptr`
- `go_archive_zip_writer_create_header(w, h) -> ptr`
- `go_archive_zip_writer_create_raw(w, h) -> ptr`
- `go_archive_zip_writer_write(w, p) -> GoIoResult`
- `go_archive_zip_writer_close_file(w) -> ptr`
- `go_archive_zip_writer_close(w) -> ptr`
- `go_archive_zip_writer_set_offset(w, n) -> void` (panics 77453 if count != 0)
- `go_archive_zip_writer_flush(w) -> ptr` (no-op nil)
- `go_archive_zip_writer_set_comment(w, comment) -> ptr`
- `go_archive_zip_writer_copy(w, r, f) -> ptr`

Corpus example:

- `go_archive_zip_demo_roundtrip() -> bool` — writes a three-file Store
  archive (file + directory + SetModTime file) into an in-memory
  multi-writer sink, reads it back, verifies names, contents, the
  directory flag, the comment, the ModTime, and the CRC, then exercises
  Writer.Copy into a second archive. Check-verified only (see
  WAVE2-ARCHIVE-ZIP-170.md § Verification — not executed; build-exe is
  blocked by (lxxii)/(lxxiii)).

## Private helpers (43 functions)

- Errors: `__go_zip_is_nil`, `__go_zip_is_eof`,
  `__go_zip_is_unexpected_eof`, `__go_zip_err_long_name`,
  `__go_zip_err_long_extra`, `__go_zip_err_comment_len`,
  `__go_zip_err_writer_closed`, `__go_zip_err_write_closed_file`,
  `__go_zip_err_file_closed_twice`, `__go_zip_err_write_dir`,
  `__go_zip_err_comment_too_long`, `__go_zip_err_dup_header`
- Bytes: `__go_zip_slice`, `__go_zip_append`, `__go_zip_le16`,
  `__go_zip_le32`, `__go_zip_le64`, `__go_zip_put_u16`,
  `__go_zip_put_u32`, `__go_zip_put_u64`
- `__go_zip_detect_utf8` (Go's detectUTF8)
- Time: `__go_zip_days_from_civil`, `__go_zip_msdos_to_time`,
  `__go_zip_time_to_msdos`
- `__go_zip_parse_extras` (zip64 / NTFS / Unix / ext-time)
- Path: `__go_zip_is_local`, `__go_zip_contains_backslash`
- Directory end: `__go_zip_find_signature`,
  `__go_zip_find_directory64_end`, `__go_zip_read_directory64_end`,
  `__go_zip_read_directory_end`
- `__go_zip_read_dir_header`, `__go_zip_zero_file`,
  `__go_zip_new_reader_inner`, `__go_zip_empty_reader`
- File open: `__go_zip_find_body_offset`, `__go_zip_empty_file_reader`,
  `__go_zip_empty_raw_reader`
- Writer: `__go_zip_writer_write_bytes`,
  `__go_zip_writer_write_header`, `__go_zip_writer_write_descriptor`,
  `__go_zip_writer_prepare`, `__go_zip_min_u64`

## Structs (17)

`GosZipFileHeader` (incl. `modified_set: bool` — mirrors Go's
`Modified.IsZero()` for CreateHeader), `GosZipFile`, `GosZipDirEntry`,
`GosZipReader`, `GoZipReaderResult`, `GosZipFileReader`,
`GoZipFileReaderResult`, `GosZipRawReader`, `GoZipRawReaderResult`,
`GoZipHeaderResult`, `GosZipWriterState`, `GosZipWriter`,
`GosZipDirEnd`, `GoZipDirHeaderResult`, `GoZipExtraResult`,
`GoZipMsDos`, `GoZipUtf8`.

## Constants (31)

Signatures (local/central/EOCD/zip64-EOCD/zip64-locator/data-descriptor/
digital), methods (Store 0, Deflate 8), flags (encrypted 0x1,
descriptor 0x8, UTF-8 0x800), versions (20/45/46), creators (Unix
0x300, FAT 0), extra ids (zip64 0x0001, NTFS 0x000a, Unix 0x000d,
ext-time 0x5455, Info-ZIP Unix 0x5855), limits (u16max/u32max,
header lengths), `GO_ZIP_PANIC_SET_OFFSET` (77453).

## Panic codes (gostd block 77400–77499)

- 77453 LIVE: `Writer.SetOffset` after data written (Go panics the same way).
- 77454–77464 RESERVED for omitted surface: 77454 FileInfoHeader,
  77455 (FileHeader).FileInfo, 77456 (FileHeader).Mode,
  77457 (FileHeader).SetMode, 77458 OpenReader, 77459 Reader.Open /
  fs.FS surface, 77460 Reader.RegisterDecompressor,
  77461 Writer.RegisterCompressor, 77462 package RegisterDecompressor,
  77463 package RegisterCompressor, 77464 Writer.AddFS.
- Verified free corpus-wide 2026-11-28 (used: 77400–77418, 77420–77435,
  77439–77443, 77446, 77447, 77451, 77452, 77499; tar reserves
  77444/77445/77448 + spares 77449/77450; 77419 slices, 77433 cmp).

## Error target ids (gostd block 900000–900099)

- 900057 ErrFormat, 900058 ErrAlgorithm, 900059 ErrChecksum,
  900060 ErrInsecurePath, 900061 errLongName, 900062 errLongExtra,
  900063 invalid comment length, 900064 writer closed twice,
  900065 write to closed file, 900066 file closed twice,
  900067 write to directory, 900068 Writer.Comment too long,
  900069 invalid duplicate FileHeader.
- Reused: 900010 io.EOF, 900011 io.ErrUnexpectedEOF.
- Verified free corpus-wide 2026-11-28 (staging max was 900046;
  tar claims 900047–900056).

## Omitted surface (no stubs; codes reserved above)

- `FileInfoHeader`, `(FileHeader).FileInfo`, `(FileHeader).Mode`,
  `(FileHeader).SetMode` — need io/fs FileInfo/FileMode.
- `OpenReader` — needs os.
- `Reader.Open` + whole fs.FS surface (`initFileList`, `openLookup`,
  `openReadDir`, `openDir`, `fileListEntry`) — need io/fs.
- `Reader.RegisterDecompressor`, `Writer.RegisterCompressor`, package
  `RegisterDecompressor`/`RegisterCompressor` — take function values;
  closure→fn_ptr adaptation is impossible in current SLA
  (LANG-PROPOSAL-closure-fnptr.md).
- `Writer.AddFS` — needs io/fs.
- `Writer.CreateHeader` duplicate-FileHeader check skipped (Go compares
  pointers — unrepresentable in SLA values).

## Divergences (see file header for full text)

- D0: `NewReader` takes the archive bytes (`Vec<u8>`) directly — gostd
  has no ReaderAt handle; `go_io_read_full` is the random-access primitive.
- D1: GoTime has no zero sentinel and no location. A zero MS-DOS
  date/time decodes to 1980-01-01 (unix 315532800), not a zero time;
  Modified keeps UTC (Go attaches a fixed-offset zone estimated from the
  extended-timestamp delta — the instant is identical, only Location differs).
- D2: Deflate (method 8) is a documented seam, NOT implemented:
  `File.Open` on a Deflate entry and `CreateHeader` with Method=8 return
  ErrAlgorithm (900058). Store + CreateRaw/Copy work fully.
- D2b: `Writer.Create` defaults to Store in gostd (Go defaults to Deflate,
  unimplemented until compress/flate lands).
- D3: `Writer.Flush` is a no-op returning nil (GosIoWriter has no buffering).
- D4: Read errors are sticky per Go's checksumReader.
- D5: `SetOffset` panics 77453 when data has been written (only live panic).
- D6: Headers are values, not pointers. Go mutates the caller's FileHeader
  in CreateHeader even on failure; gostd leaves the caller's header
  unchanged on error. A failed local-header write leaves no dir entry and
  no open file (Go appends to dir before writing the header).

## Seams used (landed corpus)

- io.sla: `GoIoResult`, `GoIoBytes`, `go_io_read_full`, `GosIoWriter`,
  `go_io_writer_write`, `go_io_writer_from_multi`,
  `go_io_multi_writer`, `go_io_eof`, `go_io_err_unexpected_eof`,
  ids 900010/900011.
- crc32.sla: `go_crc32_checksum_ieee`, `go_crc32_ieee_table`,
  `go_crc32_update`.
- time.sla: `GoTime`, `go_time_unix`, year/month/day/hour/minute/second
  + `go_time_unix_sec` accessors.
- errors.sla: `go_errors_new_with_id`, `go_errors_is`, `go_errors_nil`.
- utf8.sla: `go_utf8_decode_rune`, `go_utf8_valid_rune`,
  `go_utf8_rune_error`.
- bytes.sla: `go_bytes_clone`, `go_bytes_equal`.
- path.sla: `go_path_clean` (for `__go_zip_is_local`).

## Verification (see WAVE2-ARCHIVE-ZIP-170.md for the full record)

- 30-file splice `sa check` PASS (rc=0) on the fresh 603232ac plugin
  (iso-home), `sa` = `/media/vdb/r3base-008e66ff/zig-out/bin/sa`
  (0.1.11-优化 — the canonical `/content/sa_all/sci/zig-out/bin/sa`
  was absent during a B2-line rebuild; see the report).
- Installed plugin 0.1.8 control: fails `UseAfterMove: raw` in fmt
  (file 5 of 30), as expected — never reaches zip.
- Single-file check: fails at the first struct-literal return
  (line 1604, R1 artefact — struct undefined standalone), as expected.
- Runtime (demo execution, Go-oracle byte comparison): NOT run —
  `sa test` is broken env-wide and build-exe is blocked by
  (lxxii)/(lxxiii). Check-level verification only.
