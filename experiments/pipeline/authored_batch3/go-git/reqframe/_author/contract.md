# Contract — reqframe

Hidden suite: `tests/hidden/plumbing/protocol/packp/reqframe_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
A `stubArgs` `CommandArgs` (two arg lines, drains to flush) exercises the args
hook. Packet kinds distinguished by `pktline.Scanner.Len()` (0=flush, 1=delim).

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `(&CommandRequest{}).Encode` produces exactly `"0000"` — one flush, nothing else. |
| TestDetail02 | 2 | doc | Non-empty encode = 6 packets: `command=ls-refs\n`, caps (contains `agent`), delim (Len 1), the two arg packets, flush (Len 0) — in that order. |
| TestDetail03 | 3 | doc | A flush as first packet decodes with `Command == ""`, no error. |
| TestDetail04 | 4 | partially | A first line `bogus=ls-refs` fails decode. |
| TestDetail05 | 5 | partially | A flush where the delim belongs fails decode. |
| TestDetail06 | 6 | partially | With no Args installed, a data packet after the delim fails decode. |
| TestDetail07 | 7 | no | SHAPE: `Decode` on empty input returns within a timeout leaving `Command == ""` — no hang, no spurious command. |
| TestDetail08 | 8 | partially | Git-proto encode = ONE packet containing `git-upload-pack /repo.git`, the host, and both params; with empty Host no `host=` appears. NUL delimiting asserted via decode tests below rather than byte positions. |
| TestDetail09 | 9 | doc | Encode rejects (with `ErrInvalidGitProtoRequest` and zero bytes written): empty command, `\n` in pathname, `0x1b` in host, `0x7f` in a param. |
| TestDetail10 | 10 | no | SHAPE: for each of three degenerate payloads (no NUL, empty, no space) decode returns within a timeout and never leaves a non-empty Command from a failed parse. Which early-exit outcome fires is left free. |
| TestDetail11 | 11 | no | SHAPE: both `host=example.com` and bare `example.com` after the first NUL decode successfully and the value `example.com` survives somewhere (Host or a param) — the Host-vs-param choice is the non-derivable bit. |
| TestDetail12 | 12 | partially | `\x00\x00p1\x00\x00` decodes to `ExtraParams == ["p1"]` — doubled terminators don't create empty entries. |
| TestDetail13 | 13 | partially | A `0x01` byte in a decoded pathname fails with `ErrInvalidGitProtoRequest` — symmetric validation. |

Refusals/softening: line 10's four distinct outcomes are asserted as "prompt return + no
half-parse leaks into Command"; line 11 asserts value survival, not which field receives it;
line 8 asserts the single-packet shape and field presence, not literal NUL positions.
