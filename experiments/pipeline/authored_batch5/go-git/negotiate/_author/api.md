# Exported API — negotiate

Package `plumbing/transport` — the fetch negotiation phase:
`NegotiatePack` (v0/v1 have/want round-tripping), `ReconcileObjectFormatV2`
(v2 algorithm alignment), plus internals `nextFlush`, `applyServerACKs`,
`isSubset`, `readShallows`.

Kept visible: flush constants (`initialFlush`, `pipeSafeFlush`,
`largeFlush`, `maxInVein`), the ReconcileObjectFormatV2 doc comment,
`FetchRequest` and `ErrNoChange`/`ErrFilterNotSupported`/
`ErrShallowNotSupported` from their own files.

Callers: `remote.fetch`, v2 fetch session. In-tree tests removed: 1
(`negotiate_test.go`).
