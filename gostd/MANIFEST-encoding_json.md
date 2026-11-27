# MANIFEST — gostd encoding/json (task 170, wave 1b)

Writer line: encoding/json package. Write-first, no compilation, no tests.
Oracle: Go 1.27.2 stdlib `src/encoding/json` (read 2026-11-27):
`encode.go` (1344 lines), `decode.go` (1302), `stream.go` (526),
`scanner.go` (612), `indent.go` (184), `fold.go` (50), `tables.go`
(220), `tags.go` (40) — 4,278 lines total.
Census: CENSUS-170.md rev1 §7.2.1 — encoding/json, 1b package with
the prerequisite "encoding package interfaces first", concrete-type
subset: string/int/bool scalars, []T, map[string]T, struct fields of
those types, Marshal/Unmarshal concrete-type generation paths;
any/interface{} and generic reflection-style encode/decode NOT in
the first wave (need a reflection surface, base gap); float types
wait for float to land. jsontext / v2 are separate packages
(§7.2.2), not here.

## Files

- VM: `~/workspace/task170-work/wave1b/encoding_json.sla` — 5,563 lines,
  236 functions (140 public + 96 private helpers), 25 structs, 113 consts
- VM: `~/workspace/task170-work/wave1b/MANIFEST-encoding_json.md` (this file)
- sha256 of `encoding_json.sla`:
  a7081ecdfd684cb6798462ba35639cd2cd312a9468a909e5b92f6bb31d382937
  (this manifest's own hash is not recorded inside itself.)

**Stability note — read before trusting the sha.** This file was
still being written while this manifest was prepared: the parent
brief listed 2,034 lines; the file measured 3,867 lines, then 4,819,
then 4,816 (sha `29384b57…`, stable ~2 min) — and then the writer
line appended the whole streaming layer + refuse helpers, landing
at 5,563 lines, sha `bd579b7a…`, stable ~2 min (08:00–08:02 UTC)
before this manifest was re-issued. The sha above is the FINAL
file as of that window. The first draft of this manifest (written
against the 4,816-line snapshot) is superseded — its TODOs 1–2
(streaming functions missing, refuse helpers missing) were
resolved by the writer's late addition, and its counts are wrong
for the final file. If the writer line touches
`encoding_json.sla` again, this sha is stale and the manifest
must be re-issued against the new file.

## Functions covered vs census scope

Census scope: the concrete-type subset above, `go_encoding_json_*`.
The file is organised in the header's three layers:

- **(A) Primitives** the frontend-emitted concrete code calls:
  scanner state machine, checkValid / Valid, appendString / quote,
  HTMLEscape / Compact / Indent, parseTag / tagOptions.Contains,
  foldName, isValidNumber, unquoteBytes, the decode-state machine,
  and the marshal framing helpers.
- **(B) Complete concrete paths** for the census subset, usable
  directly and as the worked reference for layer (A): Marshal /
  Unmarshal (+ MarshalIndent) for string, int (i64), uint (u64),
  bool, []byte (base64), json.Number, []string, []int, []bool,
  map[string]string, map[string]int, map[string]bool, RawMessage,
  plus the streaming Encoder / Decoder over a byte buffer for
  those same types (+ Token / More / InputOffset / Buffered).
- **(C) Structs are frontend-emitted**: the field list, tags and
  fold matching come from the frontend; this file supplies field
  framing, tag parsing, case-insensitive matching, omitempty /
  omitzero emptiness, the ,string quoted scalar forms, and the
  decode-state primitives for the object loop (the object-loop
  recipe is spelled out in the comment on
  `go_encoding_json_decode_new`).

Public functions (140), by group:

- Error constructors (2): `go_encoding_json_syntax_error`,
  `go_encoding_json_type_error`.
- Scanner (6): `go_encoding_json_scanner_new`,
  `go_encoding_json_scanner_reset`, `go_encoding_json_scan_step`,
  `go_encoding_json_scanner_eof`, `go_encoding_json_check_valid`,
  `go_encoding_json_valid`.
- String encoding (4): `go_encoding_json_append_string`,
  `go_encoding_json_quote_string`,
  `go_encoding_json_append_string_quoted`,
  `go_encoding_json_html_escape`.
- Compact / Indent (4): `go_encoding_json_append_compact`,
  `go_encoding_json_compact`, `go_encoding_json_append_indent`,
  `go_encoding_json_indent`.
- Tags (2): `go_encoding_json_parse_tag`,
  `go_encoding_json_tag_contains`.
- Fold (2): `go_encoding_json_fold_name`,
  `go_encoding_json_field_name_matches`.
- Validation (2): `go_encoding_json_is_valid_tag`,
  `go_encoding_json_is_valid_number`.
- Omitempty / omitzero (5): `go_encoding_json_is_empty_string`,
  `go_encoding_json_is_empty_int`, `go_encoding_json_is_empty_uint`,
  `go_encoding_json_is_empty_bool`, `go_encoding_json_is_empty_len`.
- Marshal framing (13): `go_encoding_json_marshal_begin_array`,
  `go_encoding_json_marshal_end_array`,
  `go_encoding_json_marshal_begin_object`,
  `go_encoding_json_marshal_end_object`,
  `go_encoding_json_marshal_sep`, `go_encoding_json_marshal_field`,
  `go_encoding_json_marshal_map_key`,
  `go_encoding_json_marshal_null_value`,
  `go_encoding_json_marshal_bool_value`,
  `go_encoding_json_marshal_int_value`,
  `go_encoding_json_marshal_uint_value`,
  `go_encoding_json_marshal_string_value`,
  `go_encoding_json_marshal_bytes_value`.
- Map key sorting (3): `go_encoding_json_sort_string_entries`,
  `go_encoding_json_sort_int_entries`,
  `go_encoding_json_sort_bool_entries`.
- Marshal concrete paths (15): `go_encoding_json_marshal_bool`,
  `go_encoding_json_marshal_int`, `go_encoding_json_marshal_uint`,
  `go_encoding_json_marshal_string`,
  `go_encoding_json_marshal_string_no_escape_html`,
  `go_encoding_json_marshal_bytes`, `go_encoding_json_marshal_null`,
  `go_encoding_json_marshal_number`,
  `go_encoding_json_marshal_string_slice`,
  `go_encoding_json_marshal_int_slice`,
  `go_encoding_json_marshal_bool_slice`,
  `go_encoding_json_marshal_string_map`,
  `go_encoding_json_marshal_int_map`,
  `go_encoding_json_marshal_bool_map`,
  `go_encoding_json_marshal_raw_message`.
- MarshalIndent (12): `go_encoding_json_marshal_indent_bool`,
  `go_encoding_json_marshal_indent_int`,
  `go_encoding_json_marshal_indent_uint`,
  `go_encoding_json_marshal_indent_string`,
  `go_encoding_json_marshal_indent_bytes`,
  `go_encoding_json_marshal_indent_number`,
  `go_encoding_json_marshal_indent_string_slice`,
  `go_encoding_json_marshal_indent_int_slice`,
  `go_encoding_json_marshal_indent_bool_slice`,
  `go_encoding_json_marshal_indent_string_map`,
  `go_encoding_json_marshal_indent_int_map`,
  `go_encoding_json_marshal_indent_bool_map`.
- Unquote (2): `go_encoding_json_unquote_bytes`,
  `go_encoding_json_unquote`.
- Decode state (9): `go_encoding_json_decode_new`,
  `go_encoding_json_decode_read_index`,
  `go_encoding_json_decode_save_unknown_field`,
  `go_encoding_json_decode_push_field`,
  `go_encoding_json_decode_scan_while`,
  `go_encoding_json_decode_scan_next`,
  `go_encoding_json_decode_skip`,
  `go_encoding_json_decode_read_literal`,
  `go_encoding_json_decode_read_scalar`.
- Literal stores (6): `go_encoding_json_decode_literal_string`,
  `go_encoding_json_decode_literal_int`,
  `go_encoding_json_decode_literal_uint`,
  `go_encoding_json_decode_literal_bool`,
  `go_encoding_json_decode_literal_bytes`,
  `go_encoding_json_decode_literal_number`.
- Unmarshal concrete paths (13): `go_encoding_json_unmarshal_string`,
  `go_encoding_json_unmarshal_int`,
  `go_encoding_json_unmarshal_uint`,
  `go_encoding_json_unmarshal_bool`,
  `go_encoding_json_unmarshal_bytes`,
  `go_encoding_json_unmarshal_number`,
  `go_encoding_json_unmarshal_string_slice`,
  `go_encoding_json_unmarshal_int_slice`,
  `go_encoding_json_unmarshal_bool_slice`,
  `go_encoding_json_unmarshal_string_map`,
  `go_encoding_json_unmarshal_int_map`,
  `go_encoding_json_unmarshal_bool_map`,
  `go_encoding_json_unmarshal_raw_message`.
- Encoder (17): `go_encoding_json_encoder_new`,
  `go_encoding_json_encoder_set_indent`,
  `go_encoding_json_encoder_set_escape_html`,
  `go_encoding_json_encoder_encode_string`,
  `go_encoding_json_encoder_encode_int`,
  `go_encoding_json_encoder_encode_uint`,
  `go_encoding_json_encoder_encode_bool`,
  `go_encoding_json_encoder_encode_bytes`,
  `go_encoding_json_encoder_encode_number`,
  `go_encoding_json_encoder_encode_null`,
  `go_encoding_json_encoder_encode_string_slice`,
  `go_encoding_json_encoder_encode_int_slice`,
  `go_encoding_json_encoder_encode_bool_slice`,
  `go_encoding_json_encoder_encode_string_map`,
  `go_encoding_json_encoder_encode_int_map`,
  `go_encoding_json_encoder_encode_bool_map`,
  `go_encoding_json_encoder_encode_raw_message`.
- Decoder (21): `go_encoding_json_decoder_new`,
  `go_encoding_json_decoder_feed`,
  `go_encoding_json_decoder_use_number`,
  `go_encoding_json_decoder_disallow_unknown_fields`,
  `go_encoding_json_decoder_input_offset`,
  `go_encoding_json_decoder_buffered`,
  `go_encoding_json_decoder_decode_string`,
  `go_encoding_json_decoder_decode_int`,
  `go_encoding_json_decoder_decode_uint`,
  `go_encoding_json_decoder_decode_bool`,
  `go_encoding_json_decoder_decode_bytes`,
  `go_encoding_json_decoder_decode_number`,
  `go_encoding_json_decoder_decode_string_slice`,
  `go_encoding_json_decoder_decode_int_slice`,
  `go_encoding_json_decoder_decode_bool_slice`,
  `go_encoding_json_decoder_decode_string_map`,
  `go_encoding_json_decoder_decode_int_map`,
  `go_encoding_json_decoder_decode_bool_map`,
  `go_encoding_json_decoder_decode_raw_message`,
  `go_encoding_json_decoder_more`,
  `go_encoding_json_decoder_token`.
- Loud refusals (2): `go_encoding_json_refuse_marshal`,
  `go_encoding_json_refuse_unmarshal`.

Private helpers (96): `__go_json_append`, `__go_json_slice`,
`__go_json_bytes_eq`, `__go_json_hex_digit`, `__go_json_is_space`,
`__go_json_is_digit`, `__go_json_is_hex_digit`, the ten message
fragments (`__go_json_msg_cannot_unmarshal`,
`__go_json_msg_into_go_value`, `__go_json_msg_into_struct_field`,
`__go_json_msg_of_type`, `__go_json_msg_string_tag`,
`__go_json_msg_into`, `__go_json_msg_unknown_field`,
`__go_json_msg_invalid_number`, `__go_json_msg_invalid_character`,
`__go_json_msg_unexpected_eof`), `__go_json_append_type_name`,
`__go_json_append_value_desc`, `__go_json_append_scan_context`,
`__go_json_quote_char`, `__go_json_scan_error_msg`,
`__go_json_token_error_msg`, `__go_json_syntax_error_msg`,
`__go_json_scan_error`, `__go_json_scan_push_parse_state`,
`__go_json_scan_pop_parse_state`, the 31 scanner state functions
(`__go_json_state_begin_value`,
`__go_json_state_begin_value_or_empty`,
`__go_json_state_begin_string_or_empty`,
`__go_json_state_begin_string`, `__go_json_state_end_value`,
`__go_json_state_end_top`, `__go_json_state_in_string`,
`__go_json_state_in_string_esc`, `__go_json_state_in_string_esc_u`,
`__go_json_state_in_string_esc_u1`,
`__go_json_state_in_string_esc_u12`,
`__go_json_state_in_string_esc_u123`, `__go_json_state_neg`,
`__go_json_state_1`, `__go_json_state_0`, `__go_json_state_dot`,
`__go_json_state_dot0`, `__go_json_state_e`, `__go_json_state_esign`,
`__go_json_state_e0`, `__go_json_state_t`, `__go_json_state_tr`,
`__go_json_state_tru`, `__go_json_state_f`, `__go_json_state_fa`,
`__go_json_state_fal`, `__go_json_state_fals`, `__go_json_state_n`,
`__go_json_state_nu`, `__go_json_state_nul`,
`__go_json_state_error`), `__go_json_scan_step`,
`__go_json_is_safe_byte`, `__go_json_append_range`,
`__go_json_append_newline`, `__go_json_fold_rune`,
`__go_json_is_tag_punct`, `__go_json_is_ascii_letter_or_digit`,
`__go_json_hex_val`, `__go_json_getu4`,
`__go_json_decode_rune_pair`, `__go_json_null_literal`,
`__go_json_true_literal`, `__go_json_false_literal`,
`__go_json_marshal_indent_bytes`,
`__go_json_decode_read_index`, `__go_json_decode_save_error`,
`__go_json_type_error_msg`, `__go_json_decode_save_type_error`,
`__go_json_decode_save_type_error_number`,
`__go_json_string_tag_error`,
`__go_json_decode_save_string_tag_error`,
`__go_json_decode_save_literal_type_error`,
`__go_json_decode_rescan_literal`, `__go_json_less_string_entry`,
`__go_json_less_int_entry`, `__go_json_less_bool_entry`,
`__go_json_encoder_write`, `__go_json_non_space`,
`__go_json_decoder_read_value`, `__go_json_decoder_peek`,
`__go_json_decoder_token_prepare_for_decode`,
`__go_json_decoder_token_value_allowed`,
`__go_json_decoder_token_value_end`,
`__go_json_decoder_push_token_state`,
`__go_json_decoder_pop_token_state`,
`__go_json_decoder_token_error`, `__go_json_decoder_next_value`,
`__go_json_empty_token`.

Structs (25): `GoJsonScanner`, `GoJsonBytes`, `GoJsonBytesResult`,
`GoJsonValidResult`, `GoJsonStringResult`, `GoJsonIntResult`,
`GoJsonUintResult`, `GoJsonBoolResult`, `GoJsonStringSliceResult`,
`GoJsonIntSliceResult`, `GoJsonBoolSliceResult`, `GoJsonStringEntry`,
`GoJsonIntEntry`, `GoJsonBoolEntry`, `GoJsonStringMapResult`,
`GoJsonIntMapResult`, `GoJsonBoolMapResult`, `GoJsonTag`,
`GoJsonSpan`, `GoJsonDecodeState`, `GoJsonEncoder`, `GoJsonDecoder`,
`GoJsonToken`, `GoJsonTokenResult`, `GoJsonPeek`.

Consts (113): scan opcodes (12), parse states (3), scanner states
(31), max nesting depth (10000), Decoder token states (9), token
kinds (5), type codes (12: STRING/INT/UINT/BOOL/BYTES/
STRING_SLICE/INT_SLICE/BOOL_SLICE/STRING_MAP/INT_MAP/BOOL_MAP/
NUMBER), value codes (6), scan-error contexts (29 + fallthrough),
and the two error target ids (900036/900037).

Not in this file (out of census scope for encoding/json):
encoding/json/jsontext and encoding/json/v2 (§7.2.2), the
`encoding` package interfaces (seam (a)), float encode/decode
(seam (c)), and any / interface{} decode (seam (d)).

## Splice position

The file header claims position 22 (appended after the existing
21-file splice order: errors, math, strconv, strings, fmt, sync,
time, sort, runtime, io, bytes, hash, crc32, fnv, base64, hex,
utf8, path, flag, bufio, log, context). Two cautions:

- The sibling manifests disagree on the numbers: cmp and regexp
  also claim 22nd, slices claims 23rd, sync_atomic notes the
  "21-file" order actually lists 22 names. Every wave-1b package
  claims append-at-end; the relative order among the six is an
  INTEGRATION decision, not something any one manifest can settle.
- It does not matter for this file: encoding/json calls no
  wave-1b sibling function, and no sibling calls any
  `go_encoding_json_*` function (grep-verified both directions),
  so encoding/json can sit at any end position. All of its
  code-level cross-file calls (below) are satisfied by earlier
  splice files.

## Cross-file dependencies

Code-level calls only (grep-verified on non-comment lines;
the header lists three more that exist only in comments — see
the note below):

- errors.sla — `go_errors_nil`, `go_errors_new`,
  `go_errors_new_with_id` (sentinel errors).
- strconv.sla — `go_strconv_format_int` / `format_uint` /
  `format_bool` / `parse_int` / `parse_uint` / `quote`
  (scalar text, number parsing, %q quoting), and the private
  `__go_utf8_decode` / `__go_utf8_append` (UTF-8 decode/encode
  at an offset — bufio.sla calls the same two private helpers,
  the established precedent).
- strings.sla — the private `__go_str_simple_fold` (foldName).
- sort.sla — `go_sort_vec` with a named comparator (entry
  sorting; passing a named fn as the comparator is sort.sla's
  own shape).
- bytes.sla — `go_bytes_compare` (entry key ordering).
- base64.sla — `go_base64_std_encoding` / `go_base64_encode` /
  `go_base64_decode` ([]byte as base64).
- io.sla — `go_io_eof` / `go_io_err_unexpected_eof` (streaming
  Decoder EOF errors: `__go_json_decoder_read_value`,
  `__go_json_decoder_token_prepare_for_decode`,
  `go_encoding_json_decoder_token`).
- strings.sla — `go_strings_builder_write_string`
  (`__go_json_encoder_write` writes the encoded value into
  the caller-held builder).

Header-only (named in the header's dependency list but with
ZERO call sites in code): `go_sort_strings` (sort.sla, map key
sorting — the file sorts entries with `go_sort_vec` instead).
No call is made to fmt, time, sync, runtime, math, hash,
crc32, fnv, hex, utf8.sla, path, flag, bufio, log or context.
Single import: `@import "sa_std/vec.sa"`.

## Loud rejections + panic codes claimed

gostd block 77400–77499.

| code | where | meaning |
|------|-------|---------|
| 77425 | 1 live site: `go_encoding_json_refuse_marshal` | Marshal of an unsupported type (any / interface{} / float / chan / func / complex / reflection-style encode). The frontend refuses these at the call site; this helper is the loud backstop. |
| 77426 | 36 live `panic(77426)` sites | Unmarshal type mismatch / invalid syntax backstop: `decode_skip` (1), `decode_read_scalar` (1), the literal stores (13 — one per store's unreachable-unquote / bad-first-byte branch, incl. the string/int/uint/bool/bytes/number stores), the Unmarshal slice/map loops (18 — the `!= SCAN_BEGIN_ARRAY/OBJECT` and `!= SCAN_ARRAY_VALUE/OBJECT_VALUE/OBJECT_KEY` phase checks), `go_encoding_json_decoder_token` (2 — empty value / bad unquote), `go_encoding_json_refuse_unmarshal` (1). The object-loop recipe in the comment on `decode_new` also cites 77426 four times (comments, not code). |
| 900036 | `GO_JSON_SYNTAX_ERROR_ID` const + `go_encoding_json_syntax_error` | json SyntaxError sentinel via `go_errors_new_with_id`. Compare through errors.Is (target id), never by pointer equality. The Offset is NOT carried in the GosError: every result struct carries `err_offset` alongside `err`, and checkValid returns it in `GoJsonValidResult.offset`. Message text is Go's SyntaxError message verbatim. |
| 900037 | `GO_JSON_TYPE_ERROR_ID` const + `go_encoding_json_type_error` | json UnmarshalTypeError sentinel via `go_errors_new_with_id`. Same errors.Is rule. Type names in the message are constants chosen per entry point (no reflect.Type to render); struct-field errors name the field path from the decode state's error context, exactly like Go's errorContext. |

No other function in this file panics. Nothing is a silent stub.

## Collisions found

Grep-verified before writing, across `wave1a/`,
`wave1a-sunny/`, `census/`, `drive/staging/gosla-staging/gostd/`
and `wave1b/` (all six .sla files + the five sibling manifests):

- 77425: no live use anywhere in the corpus outside this
  file — free, claimed here (1 live site:
  `go_encoding_json_refuse_marshal`).
- 77426: no live use anywhere in the corpus — free, claimed
  here (36 sites).
- 900036 / 900037: no occurrences anywhere in the corpus
  outside this file — free, claimed here.
- The other files' codes named in this file's header
  (77401 errors, 77402/77403 io/bufio, 77410–77413 sync,
  77415 bufio, 77416/77417 strings, 77418 context,
  77420/77421 + 77423 time, 77422/77424 log, 77430–77439
  sort reserved, 77440–77442 runtime, 77446/77447 fmt,
  77451/77452 math, 77429 slices, 77443 maps, 77434/77435
  sync/atomic) are all other files' — no overlap with
  77425/77426 or 900036/900037.
- Function names: no duplicates inside `encoding_json.sla`,
  and no collisions with any `^fn` name in `wave1a/*.sla` or
  `wave1a-sunny/gostd/*.sla` (verified by comm on sorted
  name lists, public and private separately).
- Struct names: no collisions with any `^struct Go*` name in
  `wave1a/` or `wave1a-sunny/gostd/` (same comm check).

## Open integration seams (must be confirmed at integration; see the file header for the full text)

1. **Encoding interfaces (a).** The `encoding` package
   interfaces (BinaryMarshaler, TextMarshaler,
   BinaryUnmarshaler, TextUnmarshaler) are NOT yet written,
   so "does this type implement Marshaler" cannot be checked
   here. Hooks are modelled as FRONTEND-EMITTED functions
   (fmt `go_host_write_stdout` / log `go_log_caller`
   precedent): when the frontend knows a concrete type
   implements json.Marshaler / encoding.TextMarshaler (or
   the Unmarshaler counterparts), it emits a call to that
   type's hook and feeds the bytes through
   `go_encoding_json_append_compact` (MarshalJSON output)
   or `go_encoding_json_append_string` (MarshalText output).
   Until the `encoding` package lands, a type the frontend
   cannot prove implements a hook is marshaled by its
   concrete default encoding. MarshalerError is therefore a
   frontend error, not one this file can construct.
2. **Streaming / io.Reader + io.Writer (b).** Encoder /
   Decoder are modelled as state machines over a byte
   buffer: GoJsonEncoder writes into a caller-held
   GosBuilder, GoJsonDecoder reads from a Vec<u8> buffer
   the frontend feeds. The actual io.Reader / io.Writer
   binding — refilling the buffer from a reader, flushing
   the builder to a writer — is FRONTEND work. Consequence:
   the decoder never slides its buffer down (SLA has no Vec
   truncate), so scanp is an absolute offset, scanned stays
   0, and the buffer grows by every fed chunk for the
   decoder's lifetime; the frontend rebuilds the decoder if
   a long-lived stream's buffer growth matters. InputOffset()
   stays exact (scanned + scanp).
3. **Float (c).** Float32 / float64 are OUT of wave 1b
   (float not landed yet). Any float value or target is a
   loud refusal: marshal panics 77425, unmarshal panics
   77426. The number grammar in the scanner still accepts
   floats (the scanner is syntax, not values), and Token
   returns a JSON number as its raw literal text so a float
   literal is observable, just not decoded into a float.
   When float lands, the float encoder and literalStore
   float case belong in this file behind the same helpers.
4. **Any / interface{} (d).** Decoding into any (Go's
   valueInterface / arrayInterface / objectInterface) is
   OUT (no reflection surface, no any value in SLA).
   Marshal of an any value and Unmarshal into an any
   target are loud refusals (77425 / 77426). The frontend
   must never emit a call into this file for an any-typed
   value.
5. **No mutable globals (e).** SLA has no mutable globals,
   so there is no scanner pool, no encodeState pool and no
   type-field cache. Every scanner, decode state, encoder
   and decoder is a value the caller creates and holds.
   Field metadata is recomputed per call by the
   frontend-emitted struct routine.
6. **Cycles (f).** Go detects marshal cycles after 1000
   nested pointers and returns UnsupportedValueError. The
   concrete paths here cannot cycle (a concrete value is a
   finite tree of Vecs), so no cycle check is needed and
   none is implemented.
7. **Struct field rules (g).** Embedded-field promotion,
   dominant-field rules, "tag wins, then shallowest, then
   first index" ordering, unexported-field skipping, and
   anonymous-struct flattening are FRONTEND decisions. The
   frontend emits the resolved field list and calls
   `go_encoding_json_marshal_field` / the decode
   primitives per field. This file does NOT re-derive field
   lists from a type.

## Divergences recorded (deliberate, in-file)

- SyntaxError offsets ride in result structs, not in the
  GosError (see the sentinel note above).
- Decoder.Token returns a JSON number as its raw literal
  text (`GoJsonToken.num`), not a float64: float is not
  landed. When float lands, Token should decode the
  literal into an f64 alongside the raw text.
- Decoder.UseNumber has no effect in wave 1b: it only
  changes decoding into any, which is out. The flag is
  carried on the decode state (`go_encoding_json_decoder_use_number`
  sets `dec.d.use_number`) so the call exists and is a
  no-op, not a missing API.
- UnmarshalTypeError messages name the Go type statically
  per entry point ("string", "int", "[]string",
  "map[string]int", ...): no reflect.Type to render, so
  the type name is a constant chosen by the function.
  Struct-field errors name the field path from the decode
  state's error context, exactly like Go's errorContext.
- The ,string error message renders the item with
  strconv.Quote (Go's %q) and the type name statically.
- Compact / Indent leave dst untouched on error (Go
  returns dst[:origLen], the same observable effect; SLA
  has no Vec truncate, so the work is done in a temp vec
  and appended only on success).
- isValidTag treats a non-ASCII rune as a letter: Go uses
  unicode.IsLetter, which needs unicode tables this file
  does not carry. The only divergence is non-ASCII
  non-letters (e.g. '«', '¡'), which Go rejects and this
  file accepts as a tag name.
- The scanner's step is an i64 state code, not a Go func
  value: SLA has no fn_ptr values for closures (S3,
  RULING-EXEC-170.md section 3); the dispatch is an
  if-chain over state codes in `__go_json_scan_step`. The
  transition table is Go's scanner.go verbatim, code for
  code.
- Decoder never slides its buffer (seam (b)).
- Marshal of an unsupported type panics 77425 instead of
  returning UnsupportedTypeError, and unmarshal into an
  unsupported target panics 77426 instead of returning
  InvalidUnmarshalError / UnmarshalTypeError: the frontend
  refuses out-of-subset types at the call site, and these
  codes are the loud backstop. The error-returning Go
  shapes remain available for in-subset failures (syntax
  errors, type mismatches on concrete values, invalid
  Number literals).
- json.Number's zero value ("") marshals as "0" (Go
  compatibility rule, encode.go stringEncoder).
- No HTML escaping is applied to map keys beyond the
  string escaping every string gets: Go sorts map keys by
  the UNESCAPED key string and escapes on write, which is
  what the entry sort + field writer here do.
- In this subset omitzero is the same test as omitempty:
  the zero value is the empty value for string / int /
  uint / bool / slice / map.
- `GoJsonDecodeState.use_number` and
  `disallow_unknown_fields` are carried on the state;
  DisallowUnknownFields errors are saved via
  `go_encoding_json_decode_save_unknown_field`
  ("json: unknown field %q").

## Oracle-verification status

- Oracle files inventoried (8 files, 4,278 lines) and
  public API cross-checked by grep against this file:
  encode.go Marshal / MarshalIndent → the 15 Marshal +
  12 MarshalIndent concrete paths (generic `any` form
  is the frontend-emitted layer (A), not a function
  here); decode.go Unmarshal → the 13 Unmarshal concrete
  paths; decode.go UnmarshalTypeError / UnmarshalFieldError
  / InvalidUnmarshalError / Number → 900037 sentinel +
  `GoJson*Result` structs + `go_encoding_json_marshal_number`
  / `go_encoding_json_unmarshal_number` (Number.Float64 is
  float, out; Number.Int64 / String are covered by the
  literal path); decode.go literalStore / value /
  valueQuoted / array / object / skip / scanNext / scanWhile
  / rescanLiteral → the decode-state primitives + literal
  stores + object-loop recipe; stream.go NewDecoder /
  NewEncoder / Encode / Decode / Token / More / Buffered /
  InputOffset / SetIndent / SetEscapeHTML / UseNumber /
  DisallowUnknownFields → covered by the Encoder / Decoder
  section (encoder_new / set_indent / set_escape_html /
  encoder_encode_*; decoder_new / feed / use_number /
  disallow_unknown_fields / input_offset / buffered /
  decoder_decode_* / more / token). The four marshal
  functions the encoder calls with an escape_html argument
  (marshal_string_slice / marshal_string_map /
  marshal_int_map / marshal_bool_map) take three parameters
  (v, is_nil, escape_html); all call sites (encoder +
  MarshalIndent wrappers) pass three. A scripted arity
  check over all 694 in-file call sites found zero
  mismatches (see Verification).
- scanner.go Valid / checkValid / reset /
  eof / pushParseState / popParseState / error → covered
  (Valid, checkValid, scanner_new/reset/eof, the state
  machine); indent.go HTMLEscape / Compact / Indent →
  covered (`go_encoding_json_html_escape` / `compact` /
  `indent` + the append_* cores); fold.go foldName /
  appendFoldedName / foldRune → covered (fold_name,
  fold_rune); tables.go safeSet / htmlSafeSet → covered
  (`__go_json_is_safe_byte`); tags.go parseTag /
  tagOptions.Contains → covered (parse_tag,
  tag_contains).
- The 8 oracle files were NOT re-read line-by-line by the
  manifest writer; the .sla header records them read
  2026-11-27 by the writer line, and the public-API
  cross-check above is the manifest writer's own
  verification. The scanner transition table and the
  message fragments were spot-checked against the oracle
  shapes (state names, contexts, error strings) while
  reading the .sla file.
- Census cross-check: CENSUS-170.md rev1 §7.2.1 table row
  for encoding/json ("待写（第一波 1b）", prerequisite
  "encoding 包接口先行", concrete-type subset
  string/int/bool scalars, []T, map[string]T, struct
  Marshal/Unmarshal generation paths, any/reflection out,
  float pending) matches what was written, except the
  prerequisite itself: the `encoding` package interfaces
  are still NOT written (seam (a)), so the hook surface
  is frontend-emitted as documented, not library-checked.

## Verification

- NOT compiled, NOT tested (write-first batch, per task brief).
- sha256 of `encoding_json.sla` (5,563 lines; 140 public + 96
  private = 236 functions; 25 structs; 113 consts;
  36 live `panic(77426)` sites; 1 live `panic(77425)` site):
  a7081ecdfd684cb6798462ba35639cd2cd312a9468a909e5b92f6bb31d382937
- Static checks run by hand (grep, no compilation):
  236 `^fn ` definitions (140 public + 96 private, matching
  the manifest lists); 25 `^struct ` definitions, no
  duplicates; no duplicate fn names inside the file and no
  collisions with `wave1a/` or `wave1a-sunny/gostd/`
  (comm on sorted name lists, public/private/structs
  separately); zero `^||` block closures in code or
  comments; zero `&&` / `||` in code (the two occurrences
  are in a comment explaining the nested-if guards —
  defect (xviii) avoided); zero TODO / FIXME in the file
  (the only "stub" mention is the header's "nothing is a
  silent stub"); single import (`sa_std/vec.sa`); the file
  ends cleanly with a complete function
  (`go_encoding_json_refuse_unmarshal`); no call to
  any wave-1b sibling and no sibling call into this file.
  Arity check (scripted, re-run 2026-11-27): all 694
  in-file call sites match their definitions' parameter
  counts — zero mismatches. (A manifest draft's TODO 1
  claimed four encoder wrappers pass 3 arguments to
  2-argument marshal functions; that was a false positive
  from a stale snapshot taken before the escape_html
  parameter was threaded through the four marshal
  functions. The four functions take (v, is_nil,
  escape_html); every call site passes three.)

## TODOs left

1. **Arity — RESOLVED, was a false positive.** A manifest
   draft claimed the four encoder wrappers
   (`encoder_encode_string_slice` /
   `encoder_encode_string_map` / `encoder_encode_int_map` /
   `encoder_encode_bool_map`) pass 3 arguments to
   2-argument marshal functions. Wrong: the four marshal
   functions take `(v, is_nil, escape_html)` — the
   parameter was threaded through before the streaming
   layer was appended — and every call site (the four
   encoder wrappers plus the four MarshalIndent wrappers)
   passes three. A scripted arity check over all 694
   in-file call sites found zero mismatches. No fix needed.
2. **Splice order among the six wave-1b packages** is an
   integration decision (see "Splice position"): cmp and
   regexp also claim 22nd, slices claims 23rd. Confirm
   the final order when the splice is rebuilt; this file
   is order-independent (no inter-1b calls).
3. **The `encoding` package interfaces prerequisite**
   (seam (a)) is still unwritten — a later wave, not a
   defect in this file.
4. **Float (seam (c))** — float encode/decode + Token's
   f64 form land with float; until then float stays a
   loud refusal (77425 / 77426).
5. **Debug-phase checks (unverified shapes, same class
   as the slices manifest's list):** the
   `go_sort_vec(v, __go_json_less_*_entry)` comparator
   shape (sort.sla's own shape, but unproven with struct
   entries); the `&Vec<GoJsonStringEntry>` in-place sort
   of a caller-held vec; `Vec::push` of a Vec<u8> element
   read out of a live slice (`out.push(elem)` after
   `read_scalar` — the element is a fresh value from the
   literal store, not a borrowed element); the
   `st.err_fields[st.err_fields_len] = name` indexed
   assignment through a borrow in `decode_push_field`
   (same shape as maps' IC-1); chained index-then-field
   reads as fn arguments (`st.err_fields[i]` inside
   `__go_json_type_error_msg` — same shape as maps'
   IC-2); the `__go_utf8_decode` / `__go_utf8_append`
   private-helper reach from a non-strconv file
   (bufio precedent, same shape); `GoJsonEncoder` built
   with a struct literal over a caller-held `GosBuilder`
   (`GosBuilder { buf: out.buf }` in `encoder_new` — the
   aliasing shape is new to the corpus); nested-struct
   field assignment through a borrow
   (`dec.d.use_number = true`,
   `dec.d.disallow_unknown_fields = true` — same class
   as maps' IC-1); `__go_json_decoder_token` calling
   `decoder_decode_string` with a swapped `token_state`
   (the state save/restore around the call).

## Resolved during manifesting (was TODO in the first draft)

- **Streaming Encoder / Decoder — RESOLVED.** The writer
  line appended the full streaming layer (lines 4817–5547)
  after the first draft: Encoder (new / set_indent /
  set_escape_html / 14 encode_* / `__go_json_encoder_write`)
  and Decoder (new / feed / use_number /
  disallow_unknown_fields / input_offset / buffered / 13
  decode_* / more / token + 11 private helpers). The
  header's layer-(B) claim is now true. No streaming gap
  remains (the TODO 1 arity claim was a false positive —
  see TODO 1).
- **Refuse helpers — RESOLVED.** `go_encoding_json_refuse_marshal`
  (`panic(77425)`) and `go_encoding_json_refuse_unmarshal`
  (`panic(77426)`) now exist at the end of the file, so
  77425 has a live call site and the frontend's loud
  backstop is callable.

## Fix — defect (lxiii) landed 2026-11-27 (Muse relay 17379)

`go_encoding_json_unmarshal_raw_message` (encoding_json.sla:4809)
built two `GoJsonBytesResult` literals without the `is_nil` field
(:4812 error path, :4815 success path); the struct has 5 fields
(v, err, err_offset, set, is_nil) and the other six
GoJsonBytesResult sites all include it. The 28-file diag check
stopped with `Missing field in struct literal: GoJsonBytesResult`.
Pre-existing writer defect, unmasked once the (lxii) fix let the
check reach encoding_json.sla — NOT a (lvi) regression. Registered
as gosla writer-side source defect (lxiii), not a compiler
regression.

Fix (2 lines): add `is_nil: false` to both literals. `false` matches
the sibling error sites on the error path; on the success path the
value is a fresh non-nil slice copy, so `false` is the semantically
right value. Before sha:
bd579b7a2b5edd545531deef05fd62666baf4d34b454ad53ec0efbf640a93914.
After sha: a7081ecdfd684cb6798462ba35639cd2cd312a9468a909e5b92f6bb31d382937
(5,563 lines, unchanged count). With this fix plus the (lxii) fix,
the 28-file diag check passes rc=0 (first full completion).
Report: LVI-POSTLAND-CHECK-170.md §4; landing report
LXII-LXIII-LANDING-170.md.
