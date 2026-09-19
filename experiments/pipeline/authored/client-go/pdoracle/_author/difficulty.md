# Why hard — pdoracle

concurrency-first: a refresh goroutine, per-scope atomic lastTS with fast-path cutover, blocking futures, monotonicity, and the subtle 'never return a future stale ts' invariant. Panics hang every txn; a wrong fast-path check floods the meta service.
