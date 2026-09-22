# Contract — negside

Hidden suite: `tests/hidden/plumbing/protocol/packp/negside_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `shallow`/`unshallow` lines + flush decode into `Shallows`/`Unshallows` lists in arrival order (interleaved input verified). |
| TestDetail02 | 2 | no | SHAPE: shallow-update and push-option decoders DISAGREE on a flush-less stream — exactly one errors, the other accepts plain EOF; both return within a timeout. Which is which not pinned. |
| TestDetail03 | 3 | no | SHAPE: a `shallow <40 chars>` line adds at most one entry; wrong-length lines (`deadbeef`, 40+extra) either error or add nothing — never silently parsed. Non-hex acceptance left free. |
| TestDetail04 | 4 | no | SHAPE: a `frobnicate <hash>` line either errors or accumulates nothing — never silently collected. Whitespace-tolerance direction left free. |
| TestDetail05 | 5 | partially | `Encode` emits all shallows then all unshallows then a flush — fixed regrouping order, exact frame contents checked. |
| TestDetail06 | 6 | no | SHAPE: two options encode to exactly two data frames + flush, each frame containing its option text — framing delimits options (whether a trailing `\n` is inside the payload not pinned). |
| TestDetail07 | 7 | partially | An invalid option mid-list makes `Encode` fail with zero bytes written — validation precedes all output. |
| TestDetail08 | 8 | partially | Options containing tab/newline/`0x01` rejected; a space-containing option accepted; a `MaxPayloadSize+1` option rejected. |
| TestDetail09 | 9 | no | A bare flush decodes successfully leaving `Options` a non-nil empty slice. |
| TestDetail10 | 10 | partially | A `bad\x01option` line fails decode with `errors.Is(err, ErrInvalidPushOption)` — same predicate as encode. |

Refusals/softening: line 2 asserts the committed disagreement itself (one errors, one
accepts) — assigning which decoder does which is the non-derivable half; lines 3–4 assert
"never silently succeeds into wrong data" rather than pinning error-vs-skip; line 6 asserts
one frame per option, not the payload's exact byte content.
