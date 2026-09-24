# Bug report

WebSocket upgrades are broken. The `Sec-WebSocket-Accept` response key is
wrong, so conforming clients refuse the handshake. `Upgrade`/`Connection`/
`Sec-Websocket-Extensions` header matching is case-sensitive and misses
comma-separated tokens, so valid upgrade requests are rejected and
permessage-deflate negotiation silently drops (or advertises the extension
when the client never offered both no-context-takeover parameters). Origin
enforcement is off: requests with no Origin header are rejected (they
should be accepted), same-origin checks pass when the port or scheme
differs, and the allowed-origins list matches on host alone ignoring
scheme+port. Host:port splitting fails on origins without an explicit port
instead of applying the scheme default.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
