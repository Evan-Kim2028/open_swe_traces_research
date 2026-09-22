# Contract — gitproto-nul

Hidden suite: `tests/hidden/plumbing/protocol/packp/gitproto_nul_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
Round-trips use `Encode`/`Decode`; raw frames are built with the intact
`pktline` package.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | `Encode` on a nil writer returns the package's nil-writer sentinel. |
| TestDetail02 | 2 | yes | A request with an empty `RequestCommand` fails to encode. |
| TestDetail03 | 3 | no | SHAPE: an ASCII control byte (0x00–0x1f or 0x7f) in command, pathname, host, or an extra parameter makes encode fail; ordinary text passes. |
| TestDetail04 | 4 | no | SHAPE: the encoded payload is `command SP pathname NUL`, then `host=<host> NUL` when host is set, then a NUL plus `param NUL` per extra parameter — asserted byte-exact on the committed layout. |
| TestDetail05 | 5 | no | SHAPE: with both host and extra parameters, the payload contains `host=<h> NUL NUL param NUL` — the extra NUL is present even after host. |
| TestDetail06 | 6 | partially | `Decode` of a flush packet or an empty line yields `io.EOF`. |
| TestDetail07 | 7 | no | SHAPE: a payload missing its trailing NUL fails to decode. |
| TestDetail08 | 8 | partially | The first space splits the command; the remaining fields split on NUL — a command containing no space keeps the whole prefix as the command. |
| TestDetail09 | 9 | no | SHAPE: the second field lands in `Host` with a `host=` prefix removed; a second field lacking that prefix still lands in `Host` unchanged. |
| TestDetail10 | 10 | no | SHAPE: empty extra-parameter fields are dropped — a double NUL yields no empty parameter. |
| TestDetail11 | 11 | yes | `Decode` rejects the same control bytes `Encode` rejects. |
| TestDetail12 | 12 | no | SHAPE: invalid requests surface the `ErrInvalidGitProtoRequest` sentinel (matched by `errors.Is`, never by message). |

Refusals/softening: line 3 probes a representative control byte per field
rather than the full byte range; the set membership is the committed shape.
Lines 4–5 assert the committed byte layout on one concrete request — the
general framing is the committed part. Line 9 asserts the strip-or-keep
shape, not validation. Line 12 checks the sentinel identity only.
