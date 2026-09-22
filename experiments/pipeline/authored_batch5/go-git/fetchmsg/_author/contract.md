# Contract — fetchmsg

Hidden suite: `tests/hidden/plumbing/protocol/packp/fetchmsg_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | no | SHAPE: encode emits want/have/shallow lines each sorted ascending by hash regardless of input slice order (sortedness asserted, literal permutation unpinned). |
| TestDetail02 | 2 | no | SHAPE: `Encode` with empty `Wants` returns a non-nil error and has written zero bytes. |
| TestDetail03 | 3 | partially | Every set flag emits a line containing its keyword (`done`, `thin-pack`, `no-progress`, `include-tag`, `ofs-delta`, `shallow`, `deepen`, `deepen-relative`, `deepen-since`, `deepen-not`, `filter`, `wait-for-done`); encoding is byte-deterministic across runs. Wire order itself not pinned. |
| TestDetail04 | 4 | partially | `DeepenRelative` emits a `deepen-relative` line; `Deepen: 7` emits `deepen 7`. |
| TestDetail05 | 5 | partially | `DeepenSince` emits `deepen-since <n>` where n is the instant's unix seconds (verified with a non-UTC input). |
| TestDetail06 | 6 | partially | `Decode` returns without error, with parsed wants intact, when the args end on flush-pkt, delim-pkt, end-of-input, or a blank line. |
| TestDetail07 | 7 | no | SHAPE: an unrecognized line between valid lines does not error; the known lines still parse. |
| TestDetail08 | 8 | no | SHAPE: `deepen-relative 99` decodes without error, sets `DeepenRelative`, and leaves `Deepen` at the value the `deepen` line gave. |
| TestDetail09 | 9 | partially | `want`/`have`/`shallow` lines with a short or non-hex hash fail decode; full-length hex object IDs decode to the right `plumbing.Hash`. |
| TestDetail10 | 10 | doc | With `maxSectionLines` lowered to 2, decoding three want lines fails. |
| TestDetail11 | 11 | doc | A response with `shallow-info` before `acknowledgments`, or `acknowledgments` repeated, fails with `*MalformedResponseError`. |
| TestDetail12 | 12 | partially | `ready` followed by a premature flush ⇒ `*MalformedResponseError`; `ready` followed by end-of-input ⇒ an error (type unpinned); acknowledgments without `ready` continuing into more sections ⇒ malformed; `ready`+delim+`packfile` and `NAK`+flush both decode clean. |
| TestDetail13 | 13 | doc | After decode reaches the `packfile` header, `Packfile` is true and the next pkt-line read from the reader is the first packfile data line (pack bytes untouched). |
| TestDetail14 | 14 | doc | Acknowledgments encode ACK lines before a single `ready`; `NAK` appears only when there are no ACKs and no ready. |
| TestDetail15 | 15 | partially | A no-packfile encode writes the acknowledgments section terminated by a flush-pkt; it errors when acknowledgments are absent, marked ready, or joined by other metadata sections. |
| TestDetail16 | 16 | no | SHAPE: a URI line's interior and trailing whitespace survive decode verbatim — only the trailing newline is stripped. |

Refusals/softening: line 3 asserts flag presence + determinism, not the fixed
wire order (doc lists fields, not order); line 6 asserts clean termination for
all four committed terminators (outcome class only, no error text); line 16
asserts whitespace preservation, not a trimming implementation.
