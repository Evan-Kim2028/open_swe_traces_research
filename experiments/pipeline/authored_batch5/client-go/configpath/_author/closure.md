# Closure — configpath

Package: `config`. File: `config/config.go` (partial).

Removed (all bodies stubbed): `ParsePath`, `TxnLocalLatches.Valid`,
`GetTxnScopeFromConfig`.

Kept: config struct definitions and defaults, `GetGlobalConfig` /
`StoreGlobalConfig` / `UpdateGlobal` (shared global plumbing used by other
surfaces).

Tests edited: `config/config_test.go` deleted — `TestParsePath` and
`TestTxnScopeValue` are dedicated to this closure. `security_test.go`
stays.
