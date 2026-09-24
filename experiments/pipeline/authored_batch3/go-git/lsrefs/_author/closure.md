# Closure — lsrefs

Package: `plumbing/protocol/packp`. File: `lsrefs.go`.

Removed (7 functions stubbed): `LsRefsArgs.Encode`, `validateRefPrefix`, `LsRefsArgs.Decode`,
`LsRefsOutput.Encode`, `LsRefsOutput.Decode`, `parseLsRefsLine`, `parseFullHash`.

Kept: types `LsRefsArgs{Peel, Symrefs, Unborn bool; RefPrefixes []string}`,
`LsRefsOutput{References}`, the `tooManyRefPrefixes` constant (65536, stays visible), shared
wire constants.

All 23 `*_test.go` files in the package deleted (`lsrefs_test.go`, `command_test.go`,
`conformance_test.go` exercise this codec).
