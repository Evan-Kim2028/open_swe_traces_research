# Exported API — renamedet

Package `plumbing/object` — rename detection: `DetectRenames(changes,
*DiffTreeOptions) Changes` groups add/delete pairs into modify entries,
plus the JGit-derived `renameDetector`, `similarityIndex` (bounded
line/block hash index) and `similarityMatrix` helpers.

Kept visible: `renameDetector`/`similarityIndex`/`keyCountPair` types,
`errIndexFull`, `keyShift`/`maxCountValue`/`maxMatrixSize` consts, all doc
comments citing JGit RenameDetector / SimilarityRenameDetector /
SimilarityIndex.

Called from `difftree.go` when `DetectRenames` option is on. In-tree tests
removed: 1 (`rename_test.go`).
