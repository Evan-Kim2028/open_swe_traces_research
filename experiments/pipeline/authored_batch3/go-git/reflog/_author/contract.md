# Contract — reflog

Hidden suite: `tests/hidden/plumbing/format/reflog/reflog_bb_test.go`
(package `reflog`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `<old> <new> Name <email> <secs> <±HHMM>` + optional `\t<msg>` decodes all fields; a tab-less line yields `Message == ""`. |
| TestDetail02 | 2 | no | SHAPE: blank lines produce no entries and no error; a final line without `\n` yields at most its own entry — total is 2 or 3, never with a phantom entry. Whether the unterminated line parses is left free. |
| TestDetail03 | 3 | no | SHAPE: a malformed middle line produces an error and no entry beyond the well-formed prefix is garbage (any returned entries carry real hashes; count ≤ well-formed lines). Partial-slice-with-error vs nil left free. |
| TestDetail04 | 4 | partially | Non-hex old hash → error with zero entries; a name containing spaces keeps field positions (`Committer.Name == "Name With Spaces"`). |
| TestDetail05 | 5 | no | SHAPE: `N <a>b@c>` either rejects the line or decodes with email containing `a>b@c` — the `>` inside angles never truncates the email. |
| TestDetail06 | 6 | no | SHAPE: a 100-char seconds field either errors or produces no entry — never silently decoded. |
| TestDetail07 | 7 | no | SHAPE: `+2460` accepted → if it decodes, the zone offset is non-zero (not silently zeroed); malformed tz shapes `+246`, `+24600`, `x2460` never decode into entries. |
| TestDetail08 | 8 | no | SHAPE: encoding a `+05:30` zone emits `+0530` in the line — zone text derived from the timestamp. |
| TestDetail09 | 9 | doc | Message `"  a\n\nb  \r\n c "` encodes ending `\ta b c\n` — newlines→spaces, runs collapsed, ends trimmed. |
| TestDetail10 | 10 | partially | A whitespace-only message encodes with no TAB anywhere and the line ends right after `+0530\n`. |
| TestDetail11 | 11 | doc | A 100 KB garbage line errors with `len(err.Error()) ≤ 512` — bounded quoting. |
| TestDetail12 | 12 | partially | `Encode(buf, nil)`, `Encode(nil, entry)`, `Decode(nil)` all return errors — no panic. |

Refusals/softening: lines 2,3,5,6,7 all assert "never silently produce wrong data" shapes
rather than pinning error-vs-skip-vs-clamp choices that the `Inferable: no` marker reserves;
line 7's exact minute value (1500) is not asserted — only that a nonzero offset results.
