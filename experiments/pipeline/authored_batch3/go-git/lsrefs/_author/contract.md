# Contract — lsrefs

Hidden suite: `tests/hidden/plumbing/protocol/packp/lsrefs_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
Wire construction uses the intact `pktline` package; frames compared via
`pktline.Scanner.Text()`.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Encode of `{Peel,Symrefs,Unborn, prefixes:[heads,tags]}` yields 5 frames in order peel, symrefs, unborn, `ref-prefix refs/heads/`, `ref-prefix refs/tags/`; an all-unset args emits none of the flag lines. Trailing-newline presence on flag lines not pinned. |
| TestDetail02 | 2 | no | SHAPE: encoding with a bad prefix (`"bad prefix"`) errors AND no `peel`/`ref-prefix` frame reaches the wire — validation precedes emission. |
| TestDetail03 | 3 | partially | Prefixes `""`, containing space, tab, newline, NUL, `0x7f` all fail encode; `"refs/heads/"` encodes fine. |
| TestDetail04 | 4 | partially | Decode skips an unrecognised line and an empty line, accepts `peel` and `ref-prefix`, terminates on flush. |
| TestDetail05 | 5 | no | SHAPE: 65537 ref-prefix lines decode promptly and the resulting list is NOT a capped 65536 (len ≠ bound — the drop-fallback, asserted as "not a silent cap"). |
| TestDetail06 | 6 | doc | `"<oid> SP refs/heads/a"` decodes to one hash reference with that name+oid. |
| TestDetail07 | 7 | partially | Encoding `[v1, v1^{}, x]` emits 2 lines: base line carries `peeled:<h2>`; `^{}` never gets its own line. |
| TestDetail08 | 8 | no | SHAPE: symbolic HEAD encodes with `symref-target:<t>`; oid position carries either the resolved target hash or the `unborn` marker — never another value; unborn case (target absent) still emits `symref-target`. |
| TestDetail09 | 9 | no | SHAPE: a bare `unborn <ref>` line produces no reference (error or empty result). |
| TestDetail10 | 10 | partially | `peeled:<oid>` attribute produces a second reference `<base>^{}` immediately after the base, with the peeled hash. |
| TestDetail11 | 11 | no | SHAPE: a line with `symref-target:` decodes to `plumbing.SymbolicReference` type even with a full oid present. |
| TestDetail12 | 12 | partially | An 8-hex oid produces no reference (error or dropped line) — no silent padding. |
| TestDetail13 | 13 | partially | Double/triple spaces and an unknown `bogus-attr:x` attribute don't break decode; the ref lands with the right name. |

Refusals/softening: line 5 asserts only that the list does not silently cap at the bound
(the drop-to-empty fallback vs. cap distinction is the committed part; whether decode also
records an error is left free). Line 9 leaves open whether the malformed line is an error or
a silent drop — only that it cannot produce a reference. Line 12 same shape treatment.
