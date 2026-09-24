# Details — deadlock

1. `Detect(sourceTxn, waitForTxn, keyHash)` registers a wait-for edge
   `sourceTxn -> waitForTxn` ONLY when adding it does not close a cycle;
   on a detected cycle the edge is not registered. Inferable: partially —
   the register-on-clean ordering is visible in the call flow.
2. Cycle check is transitive DFS over `waitForMap`: `Detect(1,2,100)` then
   `Detect(2,1,200)` returns `ErrDeadlock` whose `KeyHash` is `100` — the
   stored edge's hash, not the incoming `200`. For a chain 1->2->3,
   `Detect(3,1,30)` reports the hash of the edge pointing at the source
   (`20`, the 2->3 edge). Inferable: no.
3. `register` deduplicates identical (txn, keyHash) pairs in a txn's list;
   re-Detect of the same pair does not grow the list. Inferable: no.
4. `CleanUp(txn)` drops the txn's whole wait list; `CleanUpWaitFor` removes
   one (waitForTxn, keyHash) pair and deletes the map entry when the list
   empties. Inferable: doc.
5. `Expire(minTS)` deletes entries whose SOURCE txn is `< minTS` (strictly).
   Inferable: doc for "smaller than minTS".
6. `ErrDeadlock.Error()` formats `deadlock(<KeyHash>)`. Inferable: no —
   exact string is arbitrary, only its shape is committed.
7. All exported methods take `d.lock`; `doDetect`/`register` run under it.
   Inferable: partially.
