# Closure — modconfig

Package: `config`. Files: `modules.go`, `branch.go`, `optbool.go`, `config.go` (one func).

Removed (16 functions stubbed): `Modules.Unmarshal`, `Modules.Marshal`,
`Submodule.Validate`, `validSubmoduleName`, `isPathSep`, `Submodule.unmarshal`,
`Submodule.marshal`, `Branch.Validate`, `Branch.marshal`, `quoteDescription`,
`Branch.unmarshal`, `unquoteDescription`, `NewOptBool`, `OptBool.String`,
`OptBool.FormatBool`, `parseConfigBool`, `unmarshalSubmodules` (in config.go).

Kept: `Modules`/`Submodule`/`Branch`/`OptBool` types + constants, `IsTrue`, `IsSet`,
all error vars, `dotdotPath` regex var, `pathutil` imports' targets visible, the long
security doc comment on `validSubmoduleName`, `unmarshalBranches` and the rest of
`config.go` intact.

Tests deleted: `modules_test.go`, `branch_test.go`, `optbool_test.go`, `config_test.go`.
