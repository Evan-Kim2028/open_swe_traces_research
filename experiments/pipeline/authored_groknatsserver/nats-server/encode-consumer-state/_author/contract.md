# Contract — encode-consumer-state

Hidden suite: `tests/hidden/server/encode_consumer_state_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Assertions use only names already visible to a solver: the state/pending
record types, the package magic/version/header constants, the stack-buffer
size constant, and the surviving sibling decoder, which is exercised as the
read side of the round-trip.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | no | Byte 0 of the output equals the package's file-store magic constant. (Literal value 22 is not pinned; the named constant is.) |
| TestDetail02 | 2 | no | Byte 1 equals the package's current-version constant. (Literal value 2 is not pinned; the named constant is.) |
| TestDetail03 | 3 | partially | After the header, five uvarints appear in the committed order: ack-floor consumer, ack-floor stream, delivered consumer, delivered stream, pending-map length — verified with the field values the test planted. |
| TestDetail04 | 4 | yes | With an empty pending map, the field after the zero pending length is the redelivered length and the buffer ends — no base timestamp is emitted. |
| TestDetail05 | 5 | doc | With a non-empty pending map, a signed varint immediately after the pending length decodes to the current wall-clock time at second resolution (within the call's own before/after bound). |
| TestDetail06 | 6 | no | A two-entry pending map round-trips through the surviving decoder exactly — keys, sequences, and second-resolution timestamps — and a structural parse shows each record occupies exactly three varints between the base timestamp and the redelivered length. The delta/inversion arithmetic is asserted only through the decoder's reconstruction, not by reimplementing it. |
| TestDetail07 | 7 | doc | With both maps empty, a uvarint still follows the pending length and the encoding ends after it — the redelivered length is always written. |
| TestDetail08 | 8 | partially | A two-entry redelivered map round-trips through the decoder exactly, and a structural parse shows the redelivered length followed by exactly two uvarints per entry. |
| TestDetail09 | 9 | yes | A multi-entry pending map plus multi-entry redelivered map round-trips with every entry preserved — correctness cannot depend on emission order. |
| TestDetail10 | 10 | yes | A full structural parse (header, four floors, pending length, optional base timestamp, three-varint records, redelivered length, two-varint records) consumes the returned slice exactly — no trailing slack. |
| TestDetail11 | 11 | partially | With both maps empty and large sequence values (worst-case uvarints), the encoding is no longer than the package's header-buffer constant. Only the derivable bound is asserted; nothing is asserted about allocation behaviour. |

Refusals/softening: lines 1–2 pin the named constants, not literals. Line 6
is asserted as a decoder round-trip plus a three-varints-per-record shape —
the exact delta formulas are not restated in assertions. Line 11 is asserted
only as the committed consequence (the small state fits the committed buffer
size); stack-vs-heap behaviour is not observable and is not asserted.
