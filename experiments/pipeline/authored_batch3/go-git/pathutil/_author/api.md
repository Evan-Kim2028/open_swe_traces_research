# Exported API — pathutil

Package `internal/pathutil` (module `example.internal/gitkit/v6`) — path-safety predicates
shared by index/worktree/config layers.

`IsDotGitName`, `IsHFSDotGit/Gitmodules/Gitattributes/Gitignore/Mailmap` (kept wrappers),
`IsNTFSDot*`, `IsHFSDot`, `IsNTFSDot`, `HasUnsafeComponent`, `ValidTreePath`,
`WindowsValidPath`, `ReplaceTildeWithHome`, `ErrInvalidPath`.

Callers: index/tree path validation (`ValidTreePath`), submodule name checks,
`.git`-disguise rejection, Windows materialisation checks. In-tree tests removed: 5.
