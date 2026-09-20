# Closure — pathutil

Package: `internal/pathutil`. Files: `dotgit.go`, `path_util.go`, `tree.go`, `hfs.go`,
`ntfs.go`.

Removed (10 functions stubbed): `IsDotGitName`, `ReplaceTildeWithHome`,
`HasUnsafeComponent`, `isPathSep`, `ValidTreePath`, `IsHFSDot`, `WindowsValidPath`,
`isWindowsReservedName`, `IsNTFSDot`, `asciiToLower`.

Kept: `hfsIgnoredCodepoints` table (visible), `windowsReservedNames` list (visible),
the one-line `IsHFSDotGit/Gitmodules/Gitattributes/Gitignore/Mailmap` and `IsNTFSDot*`
wrappers (they reveal the needle spellings — `gi7eba`-style short prefixes visible at
call sites), `ErrInvalidPath`, the long upstream-mirroring doc comments.

Tests deleted: all 5 in the package.
