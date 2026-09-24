# Closure — negotiate

Package: `plumbing/transport`. File: `negotiate.go`.

Removed (6 functions stubbed): `nextFlush`, `applyServerACKs`,
`NegotiatePack`, `isSubset`, `readShallows`, `ReconcileObjectFormatV2`.

Kept: flush constants, doc comments, `FetchRequest`/`ErrNoChange`/
`ErrFilterNotSupported`/`ErrShallowNotSupported` (other files), the
packp codecs (`fetchmsg`, `updreq`, `ulreq`, `srvresp` own those), the
pktline codec (`pktline`). This unit is the negotiation *policy* —
batching, ACK bookkeeping, capability selection, shallow handling —
not the wire encoding.

Tests deleted: `plumbing/transport/negotiate_test.go` (1).
