# API — sourcescodec

Module: `example.internal/msgkit/v2`, package `server`.

Excised symbols — the stream-source tracking codec:

`server/filestore.go`:
- `func (fs *fileStore) writeSourcesState() error` — encodes
  `fs.sources` (`map[string]*StreamSourceState`) into `sources.db`.
- `func (fs *fileStore) decodeSourcesState(b []byte) (uint64, error)`
  — strict decoder; returns the stamped high sequence.

`server/stream.go`:
- `func streamAndSeq(shdr string) (stream, iname string, seq uint64,
  ident string)` — parses the `Nats-Stream-Source` header value in
  BOTH layouts: legacy ack-reply form (`$JS.ACK…`) and the v2
  space-separated `"<iName> <seq> <src> <dest> <orig> [ident]"`.
- `func streamAndSeqFromAckReply(reply string)` — the legacy branch.
- `func (si *sourceInfo) genSourceHeader(orig, reply, ident string)
  string` — emits the v2 header, lifting the source sequence out of
  the inbound ack reply.

Callers: `StoreMsg`/load paths call `streamAndSeq` to track per-source
progress; `processInboundSourceMsg` stamps messages via
`genSourceHeader`; `Stop`/`recoverSourcesState` persist via
write/decodeSourcesState.

Retained scaffolding: `sourcesHeaderLen`, `errSourcesInvalidVersion`,
`recoverSourcesState`/`recoverSourcesBackwardScan` orchestration,
`parseAckReplyNum`, `sliceHeader`, `StreamSourceState`.
