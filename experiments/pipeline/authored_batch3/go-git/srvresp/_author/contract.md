# Contract — srvresp

Hidden suite: `tests/hidden/plumbing/protocol/packp/srvresp_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
Decode is run under a timeout where termination is part of the committed shape.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | `GARBAGE`, `ERR something`, `NOK` first lines all fail decode — only ACK/NAK forms are valid. |
| TestDetail02 | 2 | no | SHAPE: after `ACK…continue` + `NAK` + `ACK…ready`, decode returns promptly, no zero-hash entry exists (NAK never recorded), and ≤1 ack was collected — lines after NAK are not consumed. Whether NAK is success-vs-error is left free. |
| TestDetail03 | 3 | no | SHAPE: `ACK <h1> continue` + bare `ACK <h2>` + `NAK` decodes to exactly acks `[h1, h2]` in order. Whether the bare ACK is last-line semantics is not pinned. |
| TestDetail04 | 4 | no | `continue`/`common`/`ready` map to `ACKContinue`/`ACKCommon`/`ACKReady`; an unrecognised word either errors or lands with a status equal to none of the three constants. |
| TestDetail05 | 5 | doc | `ACKStatus(0).String() == ""`; each named constant prints non-empty. |
| TestDetail06 | 6 | partially | `ACK 1234`, `ACK `, and `ACK <40-non-hex>` all fail decode. |
| TestDetail07 | 7 | no | SHAPE: encoding an empty response emits exactly one packet and it is a data line (Len ≥ 4), not a flush/delim — the NAK literal itself not pinned. |
| TestDetail08 | 8 | no | SHAPE: encoding acks emits ≥1 frame whose first frame starts `ACK ` and carries the first hash — for both status-bearing and status-less acks. Whether remaining acks are written is left free. |
| TestDetail09 | 9 | partially | `ACK <h2> bogus` still appends — decode yields `[h1, h2]`, unrecognised-status acks are not dropped. |
| TestDetail10 | 10 | partially | A flush packet inside the response fails decode — it is not a valid terminator. |

Refusals/softening: per the `Inferable: no` lines — NAK's success/error polarity, bare-ACK
termination semantics, the empty-response literal, and emit-all-vs-emit-first are all left
free; the tests assert only the observable invariants (no NAK-as-ack, prompt termination,
one data packet, first-ack-first-frame).
