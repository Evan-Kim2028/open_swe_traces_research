# Contract — revfile

Hidden suite: `tests/hidden/plumbing/format/revfile/revfile_bb_test.go`
(package `revfile`, in-package). One `TestDetailNN` per DETAILS.md line.
`.rev` fixtures are crafted byte-exactly (RIDX|ver|hashfn|entries|packsum|checksum)
so decode is checked against the documented layout, and a hand-built
`idxfile.MemoryIndex` (2 objects: h1@offset200→pos0, h2@offset50→pos1) drives
encode. A bounded drain asserts channel-close on every path.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Valid RIDX/ver1/sha1 file decodes 2 entries; version `9` → `ErrUnsupportedVersion`; hash-fn id `7` → `ErrUnsupportedHashFunction`. |
| TestDetail02 | 2 | doc + partially | Entries arrive in file order `[3,1,2]`; channel closes on success AND on a garbage-input failure (bounded recv — open channel fails the test). |
| TestDetail03 | 3 | partially | A hash-fn id 2 file with 32-byte pack checksum + sha256 trailer decodes — the id drives checksum width and algorithm. |
| TestDetail04 | 4 | no | objCount=0 → `errors.Is(err, ErrEmptyReverseIndex)` before any entry read. |
| TestDetail05 | 5 | partially | Stored pack checksum ≠ caller's → `ErrMalformedRevFile`. (Exact-length rule is covered by TestDetail12's short-read case.) |
| TestDetail06 | 6 | no | SHAPE: two bytes after the trailing checksum → decode fails — EOF is demanded. |
| TestDetail07 | 7 | partially | Bit-flips at header byte 0, hash-fn area 10, entry 13, and pack-checksum 20 all fail — the running checksum guards every region. |
| TestDetail08 | 8 | no | SHAPE: `Encode(nil,…)` and `Encode(typed-nil *bytes.Buffer,…)` both return errors — no panic, no write. |
| TestDetail09 | 9 | no | SHAPE: `sha1.New()` produces hash-fn id 1; `sha256.New()` produces id 2 — selected by hasher size. |
| TestDetail10 | 10 | no | SHAPE: a hasher pre-fed `"dirty state"` still produces a fully decodable file — carried-in state discarded. |
| TestDetail11 | 11 | doc | Encoded entries decode as `[1, 0]` — pack-offset order (h2@50 first) mapping to index positions (h2→1, h1→0). |
| TestDetail12 | 12 | no | SHAPE: a 10-byte pack checksum (correct prefix) fails decode — short read is malformed. |

Refusals/softening: line 5's exact-length clause is covered by line 12 rather than a second
long-blob case; line 9 asserts only the two documented hasher sizes, not "every other size"
exhaustively.
