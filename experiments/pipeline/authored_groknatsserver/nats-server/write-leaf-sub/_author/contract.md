# Contract — write-leaf-sub

Hidden suite: `tests/hidden/server/write_leaf_sub_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Writes are captured in a `bytes.Buffer`; trace output is captured with the
in-package dummy test logger on a minimal client/server pair.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | An empty key writes zero bytes for both positive and non-positive counts. |
| TestDetail02 | 2 | yes | `n=1` on a plain key produces a line beginning `"LS+ foo"`. |
| TestDetail03 | 3 | doc | `n=5` on a key containing a space produces exactly `"LS+ foo bar 5\r\n"`. |
| TestDetail04 | 4 | yes | Counts encode most-significant-digit-first with no leading zeros: `1`, `10`, `999`, and the int32 maximum `2147483647` all render exactly. |
| TestDetail05 | 5 | doc | A space-free key with `n=42` produces `"LS+ foo\r\n"` — no count. |
| TestDetail06 | 6 | yes | `n=0`, `-1`, `-100` produce `"LS- foo\r\n"`; a queue key with `n=0` produces `"LS- foo bar\r\n"` — never a count on removal. |
| TestDetail07 | 7 | yes | Add, queue-add, and remove lines all end with CRLF. |
| TestDetail08 | 8 | yes | With tracing enabled: a queue add traces text containing `"foo bar 7"` (key and count), a non-queue add traces the key but not `"baz 7"`, and a remove traces the key but not a count. |

Refusals/softening: none. Line 8 asserts the committed trace payload content
(key plus count for queue adds; bare key otherwise) via substring checks on
the captured trace message — no log-line layout is pinned.
