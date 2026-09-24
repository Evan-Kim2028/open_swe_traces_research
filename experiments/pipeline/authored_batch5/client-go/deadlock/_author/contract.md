# Contract (L2) — deadlock

`Detect(sourceTxn, waitForTxn, keyHash)` registers a wait-for edge only
when adding it does not close a cycle; on a detected cycle the edge is not
registered and an `ErrDeadlock` is returned. The cycle check is transitive
over the wait-for graph, and the reported `KeyHash` is the stored hash of
the edge that points at the source — not the incoming call's hash.
Registering deduplicates identical `(txn, keyHash)` pairs so a repeated
`Detect` does not grow a txn's list. `CleanUp` drops a txn's whole wait
list; `CleanUpWaitFor` removes one `(waitForTxn, keyHash)` pair and
deletes the map entry when the list empties. `Expire` deletes entries
whose source txn is strictly smaller than `minTS`. `ErrDeadlock`'s error
text has the shape `deadlock(<keyHash>)`. All exported methods take the
detector lock, and the internal detect and register paths run under it.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `Detect` registers a wait-for edge only when it does not close a cycle |
| `TestDetail02` | the reported `KeyHash` is the stored hash of the edge pointing at the source |
| `TestDetail03` | registering deduplicates identical `(txn, keyHash)` pairs |
| `TestDetail04` | `CleanUp` drops the whole list; `CleanUpWaitFor` removes one pair and empties the map entry |
| `TestDetail05` | `Expire` removes entries whose source txn is strictly `< minTS` |
| `TestDetail06` | `ErrDeadlock.Error()` has the `deadlock(<hash>)` shape |
| `TestDetail07` | exported methods and internal detect/register run under the detector lock |
