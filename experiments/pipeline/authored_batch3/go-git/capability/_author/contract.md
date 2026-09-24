# Contract — capability

Hidden suite: `tests/hidden/plumbing/protocol/capability/capability_bb_test.go`
(package `capability`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | `EncodeList` on `a=[1,2]`, `b=[]` produces exactly `"a=1 a=2 b"` — one `name=value` token per value, bare token for a flag. |
| TestDetail02 | 2 | doc | Encoding preserves first-insertion order (`zzz,aaa,mmm` → `"zzz=1 zzz=2 aaa mmm"`); re-`Add`ing `zzz` appends its value without moving it. |
| TestDetail03 | 3 | partially | `Set` on existing name replaces values in place (`a=x b=9`); `Set` on a new name appends at end (`c=7` last). |
| TestDetail04 | 4 | no | `Add("a")` with no values on existing `a=[1,2]` leaves `Get("a") == [1 2]` — values not cleared. |
| TestDetail05 | 5 | no | Decoding `"a= b"` gives `a` exactly one value, the empty string; bare `b` gives zero values but `Supports("b")` is true. |
| TestDetail06 | 6 | partially | Decoding `"   a=1    b   "` yields exactly `[a b]` — no empty-named entry (`Supports("")==false`), re-encodes to `"a=1 b"`. |
| TestDetail07 | 7 | no | `All()` on empty list has len 0; `EncodeList` on empty and on a nil `*List` both produce zero bytes. |
| TestDetail08 | 8 | no | `Delete("b")` removes it from `Supports`; re-adding `b` puts it at the END (`"a=1 c=3 b=9"`). |
| TestDetail09 | 9 | no | `Validate` on a list containing `bogus=` fails with `errors.Is(err, ErrEmptyArgument)` — the empty-argument check precedes the unknown-name check. |
| TestDetail10 | 10 | no | Three distinct validation failures: missing arg on `agent` → `ErrArgumentsRequired`; arg on flag cap `thin-pack` → `ErrArguments`; two values on `agent` → `ErrMultipleArguments`; `symref` with two values validates; `session-id` with no arg → `ErrArgumentsRequired`. |
| TestDetail11 | 11 | no | `session-id` rejects values containing space, `0x1f`, `0x7f`, non-ASCII bytes, and a 70 000-byte value; a printable value validates. Exact bound not pinned — only that an oversized value fails. |
| TestDetail12 | 12 | no | `DefaultAgent()` equals the kept `userAgent` base; a non-blank `GO_GIT_USER_AGENT_EXTRA` produces base + separator + extra; whitespace-only extra ignored. |

Refusals/softening: line 10's full assignment table is not asserted (only the three distinct
failure modes plus two spot members, per `Inferable: no` — a solver can derive "validation
exists" but not the exact per-capability rules); line 11 pins non-printable rejection and an
oversize failure but not the exact byte bound; line 12 asserts prefix/suffix structure, not
the literal separator spelling beyond what the visible constant commits.
