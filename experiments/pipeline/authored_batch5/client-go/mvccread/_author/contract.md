# Contract (L2) — mvccread

`mvccLock.check` passes a read through when the lock's `startTS` is newer
than the read timestamp or when the lock op is a non-blocking lock kind;
a blocking write op at an older `startTS` fails. At the maximum read
timestamp with `primary` equal to the queried raw key, `check` returns a
timestamp just below the lock's own `startTS` — the writer reading its
own key — and this short-circuit requires both conditions. A lock whose
`startTS` appears in `resolvedLocks` is skipped; otherwise `check`
returns zero plus an `ErrLocked` whose key field is the raw key
mvcc-encoded at the lock version and whose fields mirror the lock.
`mvccEntry.Get` runs the lock check only under SI isolation, then returns
the first version with `commitTS <= ts` skipping rollback and lock
records, and `(nil, nil)` when nothing matches. `regionContains` is the
half-open `[startKey, endKey)` interval with an empty `endKey` meaning
unbounded. `mvccEntry.Less` orders entries by `bytes.Compare` on the
encoded keys.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `check` passes through a newer lock or a `Lock`/`PessimisticLock` op; a blocking op errors |
| `TestDetail02` | at max ts with `primary` matching the raw key, `check` returns a timestamp below the lock's `startTS`; both conditions required |
| `TestDetail03` | a resolved `startTS` skips the lock; otherwise `(0, ErrLocked)` with the key mvcc-encoded at the lock version |
| `TestDetail04` | `Get` checks the lock only under SI, returns the first `commitTS <= ts` skipping rollback/lock records, `(nil,nil)` on no match |
| `TestDetail05` | `regionContains` is `[startKey, endKey)` with empty end unbounded |
| `TestDetail06` | `Less` is `bytes.Compare` on the encoded keys |
