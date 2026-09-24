# Closure — mondecode

Package `server`, file `server/monitor.go`.

Removed (stubbed): `decodeBool`, `decodeUint64`, `decodeInt`,
`decodeState`, `decodeSubs`, `myUptime`, `redactBearerJWT`.

Retained: `ConnState`/`ConnzOptions`/`SubszOptions` types and constants,
all `Handle*z` handlers, `pollConnz` helpers, `jwt` package usage
elsewhere, `urlsToStrings`, `tlsCertNotAfter`, the entire monitoring
surface.

Import change: `strconv` blanked (used only by this closure).

Tests snipped: `TestMyUptime` (server/monitor_test.go). No test files
deleted; `TestMonitor*`, `TestConnz*`, `TestSubsz*` integration tests
still hit the decoders through HTTP and panic under the bare excision.
