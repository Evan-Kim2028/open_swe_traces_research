# Closure — hexdump

Package: `internal/logutil`. File: `internal/logutil/hex.go` (99 lines).

Removed (3 bodies stubbed): `Hex`, `hexStringer.String`, `prettyPrint`.

Kept: `hexStringer` type declaration (needed by `Hex`'s signature).

Tests: no dedicated hex test exists; callers exercise it through logging
paths.
