# Bug report — msgtrace

Message-trace header lifting is missing. `genHeaderMapIfTraceHeadersPresent`,
`getConnName`, `getCompressionType`, `sample`, and `msgTraceSupport` panic
with `excised`. Tracing never engages (the header map can never be built)
and remote-name / compression negotiation helpers panic.

Reproduce:

    go test ./server/ -run 'TestMsgTraceGenHeaderMap|TestMsgTraceConnName|TestMsgTraceBasic'

Restore: exact header parsing (key to first `:`, space/tab value trimming,
empty key/value lines dropped, stop at first malformed line), the
case-sensitive dest vs case-insensitive traceparent split, the 4-token
traceparent shape with the 0x1 sampled bit, the disabled-value
short-circuit, per-kind remote-name fallback, substring compression
mapping, and the [1..100] sampler semantics.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
