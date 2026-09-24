# API — msgrecord

Module: `example.internal/msgkit/v2`, package `server`, file
`server/filestore.go`.

Excised symbols — the on-disk message-record decode + size layer:

- `func (mb *msgBlock) msgFromBufEx(buf, sm, hh, doCopy)
  (*StoreMsg, error)` — decode one record: header, hbit/ebit flags,
  checksum, subject/hdr/msg slicing with optional copy.
- `msgFromBuf` / `msgFromBufNoCopy` — doCopy=true/false wrappers.
- `func (e errBadMsg) Error() string` — corruption error format.
- `fileStoreMsgSizeRaw(slen, hlen, mlen) uint64`, `fileStoreMsgSize`,
  `fileStoreMsgSizeEstimate` — record byte accounting.
- `isFileStoreMsgTooLarge(rl uint64) bool` — record-length sanity gate.

Callers: block load/cache paths (`loadMsgsWithLock`, `cacheLookup`),
the recovery scanners (which re-implement the same header walk
inline), `blkcodec`'s collision fallback, and `StoreMsg`/`LoadMsg`
accounting.

Retained scaffolding: `msgHdrSize`/`emptyRecordLen`/`hbit`/`ebit`/
`rlBadThresh` constants, `writeMsgRecordLocked`, the two inline
recovery scanners at ~1750/~6506, `StoreMsg`, `highwayhash`.
