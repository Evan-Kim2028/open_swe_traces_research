# Contract — ulreq-encode

Hidden suite: `tests/hidden/plumbing/protocol/packp/ulreq_encode_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
Output is framed with the intact `pktline` package; frames are compared as
decoded payloads plus frame lengths.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | Encoding a request with zero wants returns an error (non-nil; the message is not pinned). |
| TestDetail02 | 2 | partially | Two wants supplied out of order are emitted in hash-sorted order. |
| TestDetail03 | 3 | partially | A want listed twice emits a single `want` line for it. |
| TestDetail04 | 4 | yes | With capabilities set, the first want line is `want <hash> <caps>`; with none, it is `want <hash>`. |
| TestDetail05 | 5 | yes | The second and later want lines carry no capability text. |
| TestDetail06 | 6 | partially | Shallows appear after the wants as `shallow <hash>` lines, sorted; a duplicated shallow emits one line. |
| TestDetail07 | 7 | doc | `Deepen > 0` combined with `DeepenSince` or `DeepenNot` returns the `ErrDeepenMutuallyExclusive` sentinel. |
| TestDetail08 | 8 | yes | `Deepen == 0` emits no `deepen` line; `Deepen > 0` emits `deepen <n>`. |
| TestDetail09 | 9 | no | SHAPE: a non-zero `DeepenSince` emits a `deepen-since <n>` line whose number equals the Unix epoch seconds of the instant. |
| TestDetail10 | 10 | yes | Each `DeepenNot` ref produces a `deepen-not <ref>` line. |
| TestDetail11 | 11 | yes | A non-empty `Filter` produces a `filter <filter>` line. |
| TestDetail12 | 12 | yes | The last packet is a flush (zero-length) packet. |
| TestDetail13 | 13 | doc | Every non-flush payload ends with a newline — checked across requests exercising wants, shallows, deepen, deepen-not, and filter. |

Refusals/softening: line 2 asserts the emitted order only (the concrete sort
helper is an implementation detail). Line 3 asserts dedup of identical wants;
the "consecutive" qualifier means only equal-after-sort duplicates are pinned.
Line 9 asserts the committed epoch-seconds value but leaves the line's
position among the other depth lines free.
