# Bug report — optsparse

Scalar option parsing is missing. `parseDuration`,
`parseWriteDeadlinePolicy`, `trackExplicitVal`, `parseListen`,
`parseURLs`, `parseURL`, `getStorageSize`, and `parseCompression` panic
with `excised`, so `ProcessConfigFile` cannot decode durations, listen
addresses, URL lists, storage sizes, compression specs, or enum policies.

Reproduce:

    go test ./server/ -run 'TestGetStorageSize|TestListenConfig|TestListenPort|TestMalformedListenAddress|TestMalformedClusterAddress|TestParseWriteDeadline'

Restore the real semantics: string-vs-int duration compat with warnings,
host:port via SplitHostPort plus int port-only, URL dedup-as-warning with
per-entry error accumulation, K/M/G/T power-of-two storage suffixes,
compression mode/threshold forms, and error-slice accumulation (not
return) for positional config errors.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
