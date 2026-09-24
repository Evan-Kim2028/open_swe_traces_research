# Contract — reportstatus

Hidden suite: `tests/hidden/plumbing/protocol/packp/reportstatus_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Flush-first, `ok`-first, and `ng`-first streams all fail decode — `unpack <status>` must lead. |
| TestDetail02 | 2 | no | SHAPE: `unpack index-pack-failed` surfaces through `rs.Error()` as an error containing `index-pack-failed`. Whether `Decode` itself errors is left free. |
| TestDetail03 | 3 | no | SHAPE: well-formed `ok <ref>` and `ng <ref> <reason>` lines populate `CommandStatuses` in order with the reason captured. Malformed-arity enforcement left unasserted. |
| TestDetail04 | 4 | partially | `Error()` reports the unpack failure before ref failures; with ok unpack, the error names the FIRST failing ref and drops the second. |
| TestDetail05 | 5 | partially | `CommandStatus.Error()` returns nil exactly for `Status == "ok"`; statuses `""`, `"ng"`, `"failure text"` all error and carry the ref name. |
| TestDetail06 | 6 | partially | A stream ending before the flush fails decode. |
| TestDetail07 | 7 | partially | Encode emits `unpack ok\n`, `ok refs/heads/a\n`, an `ng` line carrying name+status, then flush — 4 frames, nothing else. |
| TestDetail08 | 8 | no | SHAPE: a non-ok ref encodes a line that is NOT `ok ` and contains both the ref name and its status text. The `Status=="ok"` ⇒ `ok` direction is covered by TestDetail07. |
| TestDetail09 | 9 | doc | `UnpackStatusErr.Error()` contains its status text; `CommandStatusErr.Error()` contains status and ref name; the two render differently. |
| TestDetail10 | 10 | partially | Empty stream and ref-line-first stream both fail decode. |

Refusals/softening: line 2's decode-vs-error-view split is asserted only as "the status
reaches the error view"; line 3 does not assert arity rejection (the `no` marker's
non-derivable half); line 8 asserts the negative direction only, since the ok-line form is
already pinned exactly by line 7's `doc`-level grammar.
