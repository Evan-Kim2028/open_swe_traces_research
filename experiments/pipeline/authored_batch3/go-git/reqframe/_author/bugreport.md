# Bug report

Two of the request messages this library sends to a server are framed
wrong. On the newer command channel, an empty request is written as a
garbage line instead of the empty marker, arguments are emitted before
their separator or the separator is dropped, and reading a request back
treats an early end of input as a hard failure while accepting garbage in
place of the required command line. On the transport handshake request,
fields containing injected control bytes are written straight onto the
wire, the host field comes back misaligned — shifted or merged into the
parameter list — and a request missing its terminator is accepted instead
of refused.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
