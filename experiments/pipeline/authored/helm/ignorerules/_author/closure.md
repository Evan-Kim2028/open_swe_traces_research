# Closure — ignorerules

Package: pkg/ignore. Files: rules.go.

Removed: 6 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`Empty() *Rules`; `(*Rules).AddDefaults()`; `ParseFile(file)`/`Parse(io.Reader)` -> *Rules; `(*Rules).Ignore(path, os.FileInfo) bool`.
