# Contract — binio

Hidden suite: `tests/hidden/utils/binary/binio_bb_test.go` (package `binary`, in-package).
One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `Write`/`Read` produce/consume BigEndian bytes per argument in order; a mid-sequence failure stops later arguments (write byte count; unread dest stays zero). |
| TestDetail02 | 2 | doc | Offset-VLQ decode is exact: `0x8000`→128, `0xff7f`→16511, `0x808000`→16512, `0xffff7f`→2113663, single byte `0x7f`→127. |
| TestDetail03 | 3 | doc | `WriteVariableWidthInt(128)` = `80 00` exactly; round-trip over 8 values incl. 16511/16512/2^40. |
| TestDetail04 | 4 | partially | `WriteVariableWidthInt(0)` emits exactly `00`. |
| TestDetail05 | 5 | no | SHAPE: `WriteVariableWidthInt(-1)` does not return normally — bounded writer sees >16 attempted bytes before any error return, or the call is still running at timeout; a panic or clean early return fails. |
| TestDetail06 | 6 | partially | 12 continuation bytes + terminator ⇒ error that `errors.Is` `ErrIntegerOverflow`; no wrapped value returned. |
| TestDetail07 | 7 | no | SHAPE: `ReadUntil` returns bytes before the delimiter without it; EOF-before-delimiter ⇒ empty result + non-nil error; a `*bufio.Reader` argument is consumed in place (bytes after the delimiter remain readable from it). |
| TestDetail08 | 8 | no | SHAPE: underlying read error ⇒ (empty, that error); a value delivered atomically with EOF never co-returns data+error. |
| TestDetail09 | 9 | doc | `IsBinary` reads ≤8000 bytes on clean input, reports true on a NUL at offset 100, false on short clean input. |
| TestDetail10 | 10 | no | SHAPE: `IsBinary` over a `(0,nil)` reader terminates within a timeout with (false, nil). |
| TestDetail11 | 11 | partially | Consecutive `IsBinary` calls return independent results (binary→clean→long-clean); stale buffer content is never inspected. Pool mechanics themselves not pinned (unexported, solver-renamable). |
| TestDetail12 | 12 | partially | `ReadUntil` on 256KB of delimiter-free input drains to EOF and errors — never a length-bound failure. |

Refusals/softening: line 5 asserted as non-termination shape, not a byte pattern; line 8's
"complete value lost on EOF" asserted as an invariant (error ⇒ empty, clean ⇒ complete value)
because which of the two observable outcomes fires depends on internal read mechanics not
committed in DETAILS; line 11 asserts observable independence, not the `sniffPool` symbol.
