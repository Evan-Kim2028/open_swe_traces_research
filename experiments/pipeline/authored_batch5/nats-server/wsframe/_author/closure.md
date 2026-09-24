# Closure — wsframe

Package: `server`. File: `websocket.go`.

Removed (10 funcs stubbed): `wsReadInfo.unmask`, `wsIsControlFrame`,
`wsCreateFrameHeader`, `wsFillFrameHeader`, `wsMaskBuf`, `wsMaskBufs`,
`wsIsValidCloseStatus`, `wsCreateCloseMessage`, `wsGet`,
`wsMaxMessageSize`.

Kept: `wsReadInfo`/`websocket`/`srvWebsocket`/`allowedOrigin`/
`wsUpgradeResult` structs and fields, all `ws*` opcode/bit/status/size
constants, `wsOpCode` type, `nbPoolGet`/`nbPoolPut`, the frame-read loop
(`wsReadLoop`, `wsHandleControlFrame`, `wsDecompressAndParse`, `Read`,
`ReadByte`, `nextCBuf`, `resetCompressedState`, `init`), enqueue/close
helpers (`wsEnqueueControlMessage*`, `wsEnqueueCloseMessage`,
`wsHandleProtocolError`), handshake path (`wsUpgrade`, `wsAcceptKey`,
`checkOrigin`, ...). Import `math/rand/v2` blanked.

Tests snipped in `server/websocket_test.go`: `TestWSGet`,
`TestWSIsControlFrame`, `TestWSUnmask`, `TestWSCreateCloseMessage`,
`TestWSCreateFrameHeader`. `TestWSRead*` and other frame-path tests drive
through kept code and panic under excision — left in place.
