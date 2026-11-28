# MANIFEST — compress/bzip2 (task 170 wave 2)

Package: `compress/bzip2` (Go 1.27.2). Splice position **31** (after
archive/zip). Written 2026-11-28 ~22:05–22:2x CST. **Not landed; landing
awaits Muse's dispatch. No push made.**

## Files

| file | lines | bytes | sha256 |
|---|---|---|---|
| `compress_bzip2.sla` | 1718 | 48547 | `166f8cbe1e290be84237d682d0c310b52c1f13c3039c918baf2ac19986fc8531` |
| `build31_bzip2.py` | 75 | 2893 | `8cfeb441e883404f96bc240169fed6a3853cc027b8fd5c338e60854947f54e66` |
| `splice31_bzip2.sla` (box-built) | 36116 | — | `daf1ff2e622cc21e75cd83cb4f7fef7de7d39eefdb45957ce23651c1add63de2` |

Box copies under `/media/vdb/xiaohuoche-work/task170/wave2/` verified
sha-identical to the local files (2026-11-28 ~22:2x CST).

## Oracle sources (Go 1.27.2, mirrored from the box Go tree)

`~/workspace/task170-work/wave2-oracle/compress_bzip2/`:

| file | lines | sha256 |
|---|---|---|
| `bzip2.go` | 500 | `75175949a10f63f86e0c8edbec5f96004c9d2d356a7ed7835720225a71d3cd24` |
| `bit_reader.go` | 82 | `a6e3de57d8795861d22833c5e77406cc353976d85a5072ae79cf8a5ee327daa7` |
| `huffman.go` | 234 | `c7f62572e25010a4f450cca8b9be988d1120b985f5cdcaf9cb3755916b216cd5` |
| `move_to_front.go` | 53 | `9c58462d806d97761e90c7ea3b79567abe6804b1ca4c169bbbbd91ac06576682` |

All four shas verified to match the box Go tree exactly at mirror time
(2026-11-28 ~21:44 CST).

## Public API (3)

| fn | Go source |
|---|---|
| `go_bzip2_new_reader(data: Vec<u8>) -> GosBzip2Reader` | `bzip2.NewReader` |
| `go_bzip2_reader_read(r: &GosBzip2Reader, dst: &Vec<u8>, max: i64) -> GoIoResult` | `(*Reader).Read` |
| `go_bzip2_demo_roundtrip() -> bool` | corpus example (real 65-byte bzip2 vector) |

## Helpers (48, all `__go_bzip2_*`)

Bit reader (5): `new_bit_reader`, `read_bits`, `read_bits64`,
`read_byte`, `read_bytes`. CRC (4): `crc16`, `crc32` (table build),
`crc16_update`, `crc32_update`. Errors (22): `err_structural` (20
constructors, one per distinct Go message, target ids 900070–900089)
+ `err_block_crc` + `err_file_crc`. MTF (4): `mtf_new`, `mtf_decode`,
`mtf_value`, `mtf_move_to_front`. Huffman (7): `new_huffman_tree`,
`build_huffman_node`, `huffman_decode`, `less_pair`, `less_code`,
`huffman_lengths`, `huffman_tree_len`. BWT/reader (6): `inverse_bwt`,
`reader_setup`, `reader_read_inner`, `reader_read`, `reader_read_block`,
`reader_read_from_block`. Misc (0): —

## Structs (10)

`GosBzip2BitReader`, `GosBzip2ByteResult`, `GosBzip2HuffmanNode`,
`GosBzip2HuffmanTree`, `GosBzip2HuffmanTreeResult`,
`GosBzip2BuildNodeResult`, `GosBzip2HuffmanPair`, `GosBzip2HuffmanCode`,
`GosBzip2Mtf`, `GosBzip2Reader`.

## Constants (9)

`GO_BZIP2_MAGIC` (16986), `GO_BZIP2_BLOCK_MAGIC` (54156738319193),
`GO_BZIP2_FINAL_MAGIC` (25779555029136), `GO_BZIP2_RLE_START`
(2147483647), `GO_BZIP2_REPEAT_LIMIT` (2000000),
`GO_BZIP2_MAX_SELECTORS` (18002), `GO_BZIP2_MAX_HUFFMAN_TREES` (6),
`GO_BZIP2_MAX_HUFFMAN_LENGTH` (20), `GO_BZIP2_INVALID_NODE` (-1).

## Codes claimed (corpus-wide free checks 2026-11-28 ~21:44 CST)

- Panic: **77465** (block-magic mismatch), **77466** (final-magic
  mismatch), **77467** (invalid Huffman length) — live. **77468–77472**
  reserved, not attached.
- Error target ids: **900070–900089** (20 StructuralError constructors).

## Corpus seams (all land-side, not redefined here)

`go_errors_new_with_id`, `go_errors_is`, `go_errors_nil`, `go_io_eof`,
`go_io_err_unexpected_eof`, `go_bytes_equal`, `go_sort_vec`.

## Divergences from Go (recorded in the file header, D0–D8)

- D0 NewReader takes `Vec<u8>` (no streaming handle; Go's io.Reader
  has no SLA equivalent).
- D1 u32 → u64 with 32-bit masking throughout.
- D2 preRLE aliases `tt` in Go — the reader walks `tt` directly
  (`pre_rle_len`/`pre_rle_used`).
- D3 CRC table owned per-reader (no globals).
- D4 ReadBits64 mirrors Go exactly (no sticky early return).
- D5 buildHuffmanNode guards level > 31 with the equal-symbols error.
- D6 Huffman fast path uses a mask (no variable right shift; none
  precedented in the corpus).
- D7 lengths passed by `&Vec`.
- D8 `repeat`/`repeatPower` as u64.
- Deflate-style note: none — bzip2 is self-contained; no external
  seam besides the 7 corpus functions above.
- `go_bzip2_demo_roundtrip` is check-verified only (not executed;
  build-exe is blocked by (lxxii)/(lxxiii)).

## Verification (2026-11-28 ~22:0x–22:2x CST, box 25027)

- 31-file splice `sa check` on the fresh 603232ac plugin: **rc=0 PASS**
  (`check-splice31-fresh.log`).
- Installed 0.1.8 control: rc=1, `UseAfterMove: raw` (expected, stale
  plugin) (`check-splice31-installed.log`).
- Single-file check: rc=1, parse error at :1317 — first use of a
  sibling-defined type (`GoIoResult`); R1 artefact, not a defect
  (`check-single-fresh.log`).
- Toolchain: canonical `/content/sa_all/sci/zig-out/bin/sa`
  (0.1.11-优化, rebuilt 2026-11-28 21:47 — it had been absent since
  ~21:19); fresh plugin libsla.so sha `f89fc941…` (same as the
  re-verification FINAL build); `SLA_STD_DIR` = the 603232ac plugin
  clone's `sla_std`; `SLA_NO_FRESHNESS_CACHE=1`.
