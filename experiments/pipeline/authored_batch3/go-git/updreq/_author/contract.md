# Contract — updreq

Hidden suite: `tests/hidden/plumbing/protocol/packp/updreq_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
Wire built with intact `pktline`; encoded frames read via `pktline.Scanner`.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | `Command.Action()`: zero+zero → `Invalid`; zero+id → `Create`; id+zero → `Delete`; id+id → `Update`. |
| TestDetail02 | 2 | doc | `Encode` on empty commands → `ErrEmptyCommands` with zero bytes written; `Decode` of an empty stream errors. |
| TestDetail03 | 3 | partially | `shallow <40hex>` before the command decodes into `Shallows`; `shallow deadbeef` (short) and `shallow <40 non-hex>` each fail the request. |
| TestDetail04 | 4 | no | Shallow lines + immediate flush decode as a valid request with 1 shallow, 0 commands; a bare flush with no shallows fails. |
| TestDetail05 | 5 | partially | First command without the NUL separator is rejected; with `\x00 multi_ack` the caps land in `Capabilities` (`Supports("multi_ack")`). |
| TestDetail06 | 6 | partially | `refs/heads/with space` decodes whole — name is the complete remainder including interior spaces. |
| TestDetail07 | 7 | partially | 64-hex old/new ids decode; 3-char and 2-char ids fail. |
| TestDetail08 | 8 | partially | A stream missing the terminating flush fails decode. (Trailing-payload-after-flush is the non-inferable half — not asserted.) |
| TestDetail09 | 9 | no | SHAPE: a second command `refs/heads/  padded  name` keeps its interior whitespace verbatim in `Name` — no trim/collapse. |
| TestDetail10 | 10 | partially | Encode emits shallow first, then first command containing `old new name` + NUL, then remaining commands as `<old> SP <new> SP <name>` (trailing-newline presence on later commands not pinned), then flush — 4 frames for the fixture. |
| TestDetail11 | 11 | no | SHAPE: with a non-empty capability list the first command contains a NUL followed (somewhere after it) by the caps text — the `\x00 ` leader quirk is not pinned beyond caps-after-NUL. |
| TestDetail12 | 12 | partially | A both-zero-ids command makes `Encode` fail with zero bytes emitted — validation precedes output. |

Refusals/softening: line 8 asserts only the derivable half (missing flush fails) — gold
tolerates trailing payload, so that half stays unpinned; line 11 asserts caps appear after
the first NUL without pinning the space-leader spelling; line 10 leaves later-command
trailing-newline presence free after observing both spellings.
