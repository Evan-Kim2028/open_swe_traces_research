# Bug report — mondecode

Monitoring endpoints cannot decode their query parameters. `decodeBool`,
`decodeUint64`, `decodeInt`, `decodeState`, `decodeSubs`, `myUptime`, and
`redactBearerJWT` all panic with `excised`, so every `/connz`-family
handler and the uptime formatter is broken.

Reproduce:

    go test ./server/ -run 'TestMyUptime|TestMonitor|TestConnz'

Restore the decode contract: absent params are nil-error zero values,
parse failures write a 400 and return the error, `state` accepts
open/closed/any/all case-insensitively, `subs=detail` is a distinct mode,
uptime renders `NdNhNmNs` dropping only leading zero units, and bearer
JWTs are redacted only when they actually decode as bearer claims.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
