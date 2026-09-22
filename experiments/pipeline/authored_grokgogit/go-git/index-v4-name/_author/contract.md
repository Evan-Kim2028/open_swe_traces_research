# Contract — index-v4-name

Hidden suite: `tests/hidden/plumbing/format/index/index_v4_name_bb_test.go`
(package `index`, in-package). One `TestDetailNN` per DETAILS.md line.
Output is byte-compared using the zero-hash `WithSkipHash` footer.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | A two-entry v4 index encodes to exactly header + fixed-entry + varint + suffix + NUL + footer bytes — no inter-entry padding. |
| TestDetail02 | 2 | no | SHAPE: under v2 an entry whose name field already lands on an 8-boundary still gets 8 NULs, and a shorter name gets `8 - wrote%8` NULs — asserted on total length and zero bytes. |
| TestDetail03 | 3 | doc | With deliberately unsorted input, the first emitted name is the sort-first entry and the second is compressed against it — compression runs against the sorted predecessor. |
| TestDetail04 | 4 | no | SHAPE: the first entry emits strip-length 0 plus its full name; the second emits `len(prev) - commonPrefix` then the suffix — asserted as exact wire bytes. |
| TestDetail05 | 5 | no | SHAPE: after the strip varint comes `current[prefix:]` and one terminating NUL — asserted as the exact byte region. |
| TestDetail06 | 6 | partially | A name pair sharing only the leading byte of a two-byte UTF-8 sequence compresses at the byte boundary — strip `len(prev)-1`, suffix from the second byte. |
| TestDetail07 | 7 | no | SHAPE: a writer that fails mid-second-entry leaves `lastEntry` pointing at the second entry, not the first. |
| TestDetail08 | 8 | yes | Under v2 and v3 the name field is the raw name bytes with no embedded NUL; the following NULs come from padding. |

Refusals/softening: line 1 asserts only the absence of padding (total length)
— the per-field stat layout is kept code. Line 2 asserts the committed
padding formula's observable consequences, not the pad-writing mechanism.
Line 7 asserts only which entry `lastEntry` names after a mid-entry write
failure — the error text and partial output are left free.
