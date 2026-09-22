# Bug report

The inbound protocol parser is broken. Single-shot `PING`/`PONG`/`CONNECT`/
`SUB`/`UNSUB`/`PUB` still mostly work, but anything split across two reads
corrupts or loses the operation: a `SUB` line cut in half comes back with a
garbage subject, a `PUB` payload split mid-body panics or desynchronises the
stream, and a `CONNECT` spanning a read boundary never completes. Oversized
control lines are no longer rejected — a several-megabyte `SUB` line is
accepted instead of closing the connection — and non-client (route/gateway/
leaf) links now enforce the small client limit instead of their wider bound.
Unauthenticated connections can send `PING`/`SUB` before `CONNECT` without
being refused, and `R*`/`L*`/`A*` ops from plain clients are accepted rather
than rejected as protocol errors. `HPUB`/`HMSG` messages lose their headers:
header accessors return nothing even when the wire carried a `NATS/1.0`
block. Parser error messages have lost their protocol excerpt.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
