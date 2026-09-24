# Contract — get-storage-size

Hidden suite: `tests/hidden/server/get_storage_size_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | `int64` inputs — including zero, negative, and large values — pass through unchanged with a nil error. |
| TestDetail02 | 2 | yes | Non-string, non-`int64` inputs (other int widths, float, bool, byte slice) each return a non-nil error. |
| TestDetail03 | 3 | no | The empty string returns `(0, nil)`. (The DETAILS line commits the tuple itself; asserted as-is.) |
| TestDetail04 | 4 | yes | `"7K"` returns `7<<10`; a signed prefix (`"-2K"`) parses — the numeric prefix is a base-10 int64. |
| TestDetail05 | 5 | yes | Prefixes that are not base-10 int64s (`"abc"`, `"12.5"`, empty prefix, leading space, `0x` hex) all error. |
| TestDetail06 | 6 | partially | `"2K"` returns `2<<10` — the binary-shift semantics of the K suffix. |
| TestDetail07 | 7 | partially | `"3M"` returns `3<<20` — the binary-shift semantics of the M suffix. |
| TestDetail08 | 8 | partially | `"4G"` returns `4<<30` — the binary-shift semantics of the G suffix. |
| TestDetail09 | 9 | partially | `"5T"` returns `5<<40` — the binary-shift semantics of the T suffix. |
| TestDetail10 | 10 | yes | Unsupported last characters (`P`, `B`, `x`, a digit, a trailing space) all error. |
| TestDetail11 | 11 | no | Lowercase `k`, `m`, `g`, `t` are rejected — asserted only as "an error is returned" for each lowercase suffix, with no claim about the error's text. |
| TestDetail12 | 12 | yes | A space between number and suffix (`"1 K"`, `"1  K"`, `"1024 "`) errors. |

Refusals/softening: lines 6–9 are `partially` — asserted only as the numeric
result each committed suffix produces; nothing is asserted about how the
shift is implemented. Line 11 is asserted only as rejection (error returned);
the DETAILS line gives no derivable basis for a specific error message.
