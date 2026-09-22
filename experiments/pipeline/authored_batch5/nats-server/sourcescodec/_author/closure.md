# Closure — sourcescodec

Files: `server/filestore.go`, `server/stream.go`.

Stubbed (5 symbols): `fileStore.writeSourcesState`,
`fileStore.decodeSourcesState`, `streamAndSeq`,
`streamAndSeqFromAckReply`, `sourceInfo.genSourceHeader`.

Retained scaffolding: `sourcesHeaderLen`, `errSourcesInvalidVersion`,
`sourcesStatePath`, `recoverSourcesState`, `recoverSourcesBackwardScan`,
`parseAckReplyNum`, `sliceHeader`/`genHeader` (hdrsurgery banked
separately), `sourceInfo`. No imports blanked.

Test coverage snipped (restored for gold/cheat):
`TestFileStoreSourcesDecodeRejectsMalformed` (6-case truncation table),
`TestFileStoreSourcesRecovery` (round-trip incl. restart).
