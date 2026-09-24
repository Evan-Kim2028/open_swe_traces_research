# Bug report

WebSocket framing is broken. Outbound frames have malformed headers:
payloads over 125 bytes emit wrong length encodings, the masking key is
missing or zero, and continuation frames carry the wrong opcode bits.
Inbound masked frames decode incorrectly once a frame is split across two
reads — the unmask restarts the key at position 0 mid-frame. Masking of
multi-buffer writes restarts the key per buffer instead of continuing.
Close frames are wrong: invalid status codes like 1005/1006/1015 are
accepted on receive, and generated close messages truncate the reason
incorrectly (no ellipsis, or wrong byte count) so the 125-byte control
limit is exceeded. Reads needing more bytes than the buffer holds either
hang or return partial data instead of reading the remainder from the
connection. Compressed connections blow up: the message size cap no longer
scales with max_payload.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
