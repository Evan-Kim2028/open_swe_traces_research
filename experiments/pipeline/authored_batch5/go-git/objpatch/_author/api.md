# Exported API — objpatch

Package `plumbing/object` — `Patch` construction from `Change`s, the
`fdiff.FilePatch` adapters (`changeEntryWrapper`, `textFilePatch`,
`textChunk`), the `FileStat`/`FileStats` containers and the `printStat`
diffstat printer.

`getPatch[Context]`, `filePatchWithContext`, `isSubmodule`,
`submoduleContent`, `submoduleFilePatch`, `fileContent`;
`Patch.FilePatches/Message/Encode/Stats/String`; wrapper methods;
`FileStat.String`, `FileStats.String`, `printStat`,
`getFileStatsFromFilePatches`; `ErrCanceled`.

Callers: `Change.Patch`, `Commit.Patch`, `Tree.Patch`. In-tree tests
removed: 5 (`patch_test`, `patch_stats_test`, `change_test`,
`commit_test`, `tree_test` — all reach the patch builder).
