# Closure — objpatch

Package: `plumbing/object`. File: `patch.go`.

Removed (25 functions stubbed): `getPatch`, `getPatchContext`,
`filePatchWithContext`, `isSubmodule`, `submoduleContent`,
`submoduleFilePatch`, `fileContent`; `Patch.FilePatches`, `.Message`,
`.Encode`, `.Stats`, `.String`; `changeEntryWrapper.Hash`, `.Mode`,
`.Path`, `.Empty`; `textFilePatch.Files`, `.IsBinary`, `.Chunks`;
`textChunk.Content`, `.Type`; `FileStat.String`, `FileStats.String`,
`printStat`, `getFileStatsFromFilePatches`.

Kept: `ErrCanceled`, `Patch`/`FileStat`/`FileStats` types, the wrapper
structs, all doc comments (submodule format, `diff.c` scaling link,
empty-entry rule), `DefaultContextLines` consumer wiring, and the
`fdiff` unified encoder + `utils/diff` engine as readable siblings.

Tests deleted: `patch_test.go`, `patch_stats_test.go`, `change_test.go`,
`commit_test.go`, `tree_test.go` (5 — every suite that drives the patch
builder; blob/tag/object/difftree suites stay).
