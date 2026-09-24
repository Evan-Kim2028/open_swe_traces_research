# Exported API — modconfig

Package `config` (module `example.internal/gitkit/v6`) — `.gitmodules` and `[branch]`
section handling plus the tri-state bool.

`Modules{Submodules map[string]*Submodule}` with `Unmarshal([]byte) error`,
`Marshal() ([]byte, error)`; `Submodule{Name, Path, URL, Branch}` with `Validate()`;
`Branch{Name, Remote, Merge plumbing.ReferenceName, Rebase, Description}` with
`Validate()`; `OptBool` (Unset/False/True) with `NewOptBool`, `IsTrue`, `IsSet`,
`String`, `FormatBool`. Error vars kept.

Callers: `Config` loading, submodule porcelain. In-tree tests removed: 4.
