# Closure — msgrecord

File: `server/filestore.go`

Stubbed (8 symbols): `errBadMsg.Error`, `msgBlock.msgFromBuf`,
`msgBlock.msgFromBufNoCopy`, `msgBlock.msgFromBufEx`,
`fileStoreMsgSizeRaw`, `fileStoreMsgSize`, `isFileStoreMsgTooLarge`,
`fileStoreMsgSizeEstimate`.

Retained scaffolding: record-format constants (`msgHdrSize`, `hbit`,
`ebit`, `rlBadThresh`…), `writeMsgRecordLocked` (the encoder — left
visible so the decode contract stays inferable), the inline recovery
scanners, compression layer (banked in `blkcodec`). No imports blanked.

Test coverage snipped (restored for gold/cheat):
`TestFileStoreEraseMsg`, `TestFileStoreCorruptionSetsHbitWithoutHeaders`,
`TestFileStoreConvertToEncryptedDoesNotResurrectXoredCache` (filestore_test.go).
