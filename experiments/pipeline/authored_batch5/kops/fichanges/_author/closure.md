# Closure — fichanges

Package: `upup/pkg/fi` (`example.internal/clustkit/upup/pkg/fi`).

Files: `upup/pkg/fi/changes.go` (4 funcs).

Removed functions (bodies stubbed): `BuildChanges`, `equalFieldValues`, `equalMapValues`,
`equalSlice`.

Exported entry point(s): `BuildChanges` — called by `DefaultDeltaRunMethod`/`Render` for
every fi task to decide whether an update is needed. `CompareWithID` (declared elsewhere)
is consumed here.

Test files removed in excision: `dryruntarget_test.go`.
