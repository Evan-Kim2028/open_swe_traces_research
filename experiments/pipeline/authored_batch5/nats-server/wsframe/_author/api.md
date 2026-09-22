# Exported API — wsframe

Package `server` (module `example.internal/msgkit/v2`) — WebSocket (RFC 6455)
frame codec: header construction, masking/unmasking, close-message bodies,
control-frame classification, buffered reads.

Surface: `wsFillFrameHeader(fh []byte, useMasking, first, final, compressed
bool, frameType wsOpCode, l int) (int, []byte)` — fills a caller buffer and
returns (header length, masking key or nil); `wsCreateFrameHeader(useMasking,
compressed bool, frameType wsOpCode, l int) ([]byte, []byte)` — pooled
variant returning (header, key); `wsMaskBuf(key, buf)` /
`wsMaskBufs(key, bufs)` — XOR masking, the multi-buffer variant treats the
buffers as one contiguous stream; `(r *wsReadInfo) unmask(buf)` — streaming
unmask honouring the running `mkpos` key position;
`wsIsControlFrame(wsOpCode) bool`; `wsIsValidCloseStatus(code int) bool`;
`wsCreateCloseMessage(status int, body string) []byte` (pooled, 2-byte
status + truncated body); `wsGet(r io.Reader, buf, pos, needed) ([]byte,
uint64, error)`; `wsMaxMessageSize(mpay int) uint64`.

Callers: `wsReadLoop`/`wsHandleControlFrame` (masking, `wsGet`),
`wsEnqueueControlMessageLocked`, `wsEnqueueCloseMessage`, flush paths, the
leafnode client writer, tests.
