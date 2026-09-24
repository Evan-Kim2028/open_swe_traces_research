# Closure — featflags

Package: `pkg/featureflag` (`example.internal/clustkit/pkg/featureflag`).

Files: `pkg/featureflag/featureflag.go` (5 funcs).

Removed functions (bodies stubbed): `new`, `Enabled`, `Bool`, `ParseFlags`, `Get`.

Exported entry point(s): `featureflag.X.Enabled()` guards experimental code paths across
the codebase; `ParseFlags` is the env-var entry point (called from `init` and tests).

Test files removed in excision: `featureflag_test.go`.
