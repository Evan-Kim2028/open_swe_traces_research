# Details — configpath

1. `ParsePath` requires scheme `tikv` matched CASE-INSENSITIVELY
   (`strings.ToLower(u.Scheme)`) — but the error message still says
   `expected [kvstore]`. Inferable: no — the case fold and the stale
   message are arbitrary.
2. `disableGC` query accepts `"true"`/`"false"`/`""` (case-insensitive);
   any other value errors with `"disableGC flag should be true/false"`.
   Inferable: no.
3. `keyspaceName` passes through verbatim (default ""); `etcdAddrs` is
   `strings.Split(u.Host, ",")` — an empty host yields `[""]`. Inferable:
   partially.
4. `TxnLocalLatches.Valid` errors only when `Enabled && Capacity == 0`.
   Inferable: doc.
5. `GetTxnScopeFromConfig` returns the global config's `TxnScope` when
   non-empty, overridden by the `injectTxnScope` failpoint, else
   `oracle.GlobalTxnScope` ("global"). Inferable: partially — the
   failpoint hook is discoverable, its precedence is not.
