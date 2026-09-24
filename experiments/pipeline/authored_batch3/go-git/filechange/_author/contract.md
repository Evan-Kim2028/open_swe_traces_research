# Contract — filechange

Hidden suite: `tests/hidden/plumbing/object/filechange_bb_test.go`
(package `object`, in-package). One `TestDetailNN` per DETAILS.md line.
Blobs are stored through `memory.NewStorage()`; `Tree`s are built with their
storer field set (in-package access required).

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | Empty `Change` → `Action()` errors; To-only → `merkletrie.Insert`; From-only → `merkletrie.Delete`; both → `merkletrie.Modify`. |
| TestDetail02 | 2 | no | A name-only `ChangeEntry` (no tree) counts as PRESENT: From-only name-only → `Delete`; both-sides name-only → `Modify` — full-struct equality, not presence-of-name. |
| TestDetail03 | 3 | no | Insert yields `(nil, file, nil)`; delete yields `(file, nil, nil)`; a Dir-mode entry backed by a real storer yields `(nil, nil, nil)` with no error. |
| TestDetail04 | 4 | no | `String()` of a delete contains the FROM path; `String()` of a modify with differing names contains the FROM path — output asserted by containment, not exact format. |
| TestDetail05 | 5 | partially | `sort.Sort(Changes{del(zzz), ins(aaa)})` orders `[ins, del]` — deletion sorts by its old path `zzz.txt`, after `aaa.txt`. |
| TestDetail06 | 6 | no | `Lines()` on `"a\r\nb\rc\n"` → `["a\r", "b\rc"]` — `\r` never stripped, split on `\n` only, interior `\r` does not break a line. |
| TestDetail07 | 7 | partially | `"a\nb\n"` → `["a","b"]` (no phantom tail); empty blob → zero-length slice (not `[""]`). |
| TestDetail08 | 8 | partially | `.txt` file with NUL-payload content reports `IsBinary() == true`; clean text reports false — detection is content-based. |
| TestDetail09 | 9 | partially | `String()` of a delete contains `Delete` and the path; `String()` of a malformed change contains `malformed` rather than panicking. Exact `<Action: X, Path: Y>` punctuation not pinned beyond substring presence. |
| TestDetail10 | 10 | yes | `Contents()` on a 22 KB blob returns the exact stored string — no truncation at any sniff window. |

Refusals/softening: line 9's exact render format is committed as `<Action: X, Path: Y>` but
asserted by substring (action word + path) to leave separator punctuation unpinned;
`malformed change` is asserted as the substring `malformed`. All other rows assert the
committed behaviour directly.
