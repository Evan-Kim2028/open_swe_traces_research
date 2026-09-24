# Contract (L2) — configpath

`ParsePath` parses the client endpoint URL: it requires one specific scheme
(an unrelated scheme must produce an error), splits the host list on commas
into `etcdAddrs` — an empty host yields a single empty element — passes
`keyspaceName` through verbatim (absent means `""`), and parses `disableGC`
from `true`/`false`/empty (case-insensitive), erroring on any other value.
`TxnLocalLatches.Valid` errors only when the latches are enabled with zero
capacity. `GetTxnScopeFromConfig` returns the global config's `TxnScope`
when non-empty and `oracle.GlobalTxnScope` otherwise.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `ParsePath` requires one specific scheme — an unrelated scheme errors while the accepted scheme parses |
| `TestDetail02` | `disableGC` accepts `true`/`false`/empty case-insensitively and errors on anything else |
| `TestDetail03` | `keyspaceName` passes through verbatim and `etcdAddrs` is the comma-split host list with `[""]` for an empty host |
| `TestDetail04` | `Valid` errors only when enabled with zero capacity |
| `TestDetail05` | `GetTxnScopeFromConfig` returns the configured scope or the global default |
