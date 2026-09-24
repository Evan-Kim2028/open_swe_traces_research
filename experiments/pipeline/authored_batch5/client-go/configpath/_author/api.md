# Exported API — configpath

Package `config` (module `example.internal/kvstore/v2`).

- `ParsePath(path string) (etcdAddrs []string, disableGC bool,
  keyspaceName string, err error)` — parses `tikv://` client URLs.
- `(*TxnLocalLatches) Valid() error` — latch config validation.
- `GetTxnScopeFromConfig() string` — `@@txn_scope` resolution with
  failpoint override.

Kept: `Config`/`PDClient`/`TxnLocalLatches` structs, `Default*`
constructors, `GetGlobalConfig`/`StoreGlobalConfig`/`UpdateGlobal`
(global-config plumbing owned elsewhere).

Callers: `kvclient`/rawkv client constructors call `ParsePath` on the
endpoint list; `Valid` runs at config load; `GetTxnScopeFromConfig` feeds
oracle selection. In-tree `config_test.go` (`TestParsePath`,
`TestTxnScopeValue`) removed with the closure.
