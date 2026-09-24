# Closure — unidiff

Package: `plumbing/format/diff`. File: `unified_encoder.go`.

Removed (11 functions stubbed): `UnifiedEncoder.Encode`, `writeFilePatchHeader`,
`appendPathLines`, `hunksGenerator.Generate`, `processHunk`, `addLineNumbers`,
`processEqualsLines`, `splitLines`, `hunk.writeTo`, `hunk.AddOp`, `op.writeTo`.

Kept: `UnifiedEncoder` + ctor + prefix/color setters, `DefaultContextLines`,
`operationChar`/`operationColorKey` maps (the `+`/`-`/space choice is visible),
`hunksGenerator`/`hunk`/`op` structs, `ColorConfig`.

Tests deleted: `unified_encoder_test.go` (only test in the package).
