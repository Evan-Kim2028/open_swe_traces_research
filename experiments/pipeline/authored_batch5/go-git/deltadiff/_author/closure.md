# Closure — deltadiff

Package: `plumbing/format/packfile`. Files: `diff_delta.go`,
`delta_index.go`.

Removed (16 functions stubbed): `GetDelta`, `getDelta`, `DiffDelta`,
`diffDelta`, `encodeInsertOperation`, `encodeCopyOperation`;
`deltaIndex.init`, `.findMatch`, `matchLength`, `countEntries`,
`deltaIndex.copyEntries`, `newDeltaIndexScanner`, `deltaIndexScanner.scan`,
`tableSize`, `leadingZeros`, `hashBlock`.

Kept: `s`/`blksz`/`maxCopySize`/`maxChainLength` consts (with upstream
links), the `T` hash table, `len8tab`, `deltaIndex`/`deltaIndexScanner`
type skeletons, all doc comments including the linear-complexity and
findMatch sentinel notes; `delta_selector.go` and `patch_delta.go` stay
fully implemented — the apply-side decoder is a readable sibling.

Tests deleted: `delta_test.go`, `diff_delta_fuzz_test.go`,
`delta_selector_test.go`, `delta_selector_cycle_test.go`,
`encoder_test.go`, `encoder_advanced_test.go`, `parser_test.go`,
`parser_fuzz_test.go` (8 — every suite that reaches delta generation;
scanner/promisor/patch_delta/packmeta/object_pack/internal/fsobject/
common suites stay).
