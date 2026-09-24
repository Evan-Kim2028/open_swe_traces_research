# Closure — updreq

Package: `plumbing/protocol/packp`. Files: `updreq.go`, `updreq_decode.go`, `updreq_encode.go`.

Removed (10 functions stubbed): `validateUpdateRequests`, `Command.Action`, `Command.validate`,
`UpdateRequests.Decode`, `parseCommand`, `parseHash`, `UpdateRequests.Encode`,
`UpdateRequests.encodeShallow`, `UpdateRequests.encodeCommands`, `formatCommand`.

Kept: types `UpdateRequests`, `Command`, `Action` + its constants; error vars `ErrEmptyCommands`,
`ErrMalformedCommand`, `ErrEmpty`, `errNoCommands`, `errMissingCapabilitiesDelimiter`, `errNoFlush`
and the error-formatting helpers; the shared `common.go` wire constants (`shallow`, `unshallow`,
`ack`, `nak`, `eol`, `null`, `shallowNoSp`, hex sizes) and `ErrNilWriter`.

All 23 `*_test.go` files in the package deleted (sibling suites exercise the codec through
`conformance_test.go`, `fetch_test.go`, `smart_test.go`, `updreq_*_test.go`).
