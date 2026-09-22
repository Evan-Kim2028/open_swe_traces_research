# Details — batchbuild

1. `buildWithLimit` exempts entries with `priority() >= 10`
   (`highTaskPriority`) from the limit count, and keeps draining while a
   high-priority entry remains — so `buildWithLimit(0)` still sends the
   high-priority head, and a build at `limit=2` over {pri11,pri5,pri1}
   emits all 3. Inferable: partially — the `>=10` constant and the
   "don't consume limit" rule are choices.
2. Entries with a non-empty `forwardedHost` are routed into
   `forwardingReqs[host]` (per-host `BatchCommandsRequest`) instead of
   the main request; every built entry — forwarded or not — consumes one
   `idAlloc`. Inferable: partially — the forwarding map shape is in the
   signature.
3. `reset()` calls `entries.clean()` which removes only CANCELED queue
   entries — live queued entries SURVIVE a reset, and `idAlloc` is never
   reset, so request ids are monotonic across builds. Inferable: no.
4. Canceled entries are skipped inside the build loop BEFORE `idAlloc`
   assignment — they consume no request id. Inferable: no.
5. `cancel(err)` sets `err` and closes `res` on EVERY queued entry, then
   empties the queue via `entries.reset()` (different from `clean`).
   Inferable: partially.
6. `Take`-pop order is priority order — emitted request order follows
   the heap, not insertion order. `buildWithLimit` returns a nil request
   (not an empty one) when nothing was collected. Inferable: partially.
