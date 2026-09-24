# Closure — filechange

Package: `plumbing/object`. Files: `file.go`, `change.go`.

Removed (9 functions stubbed): `File.Contents`, `File.IsBinary`, `File.Lines`,
`Change.Action`, `Change.Files`, `Change.String`, `Change.name`, `Changes.Less`,
`Changes.String`.

Kept: `File`/`FileIter`/`Change`/`ChangeEntry`/`Changes` types, `NewFile`, `FileIter`
methods, `Change.Patch/PatchContext`, `Changes.Len/Swap/Patch`, `binary.IsBinary`
helper (kept source), doc comments.

Tests deleted: `file_test.go`, `change_test.go`.
