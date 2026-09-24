# Details — priorityqueue

1. The queue is a max-heap on `Item.priority()` — `prioritySlice.Less`
   compares `>` so the LARGEST priority value pops first (the doc comment
   claiming "higher priority is the lower value" is stale). Inferable: no —
   direction is arbitrary.
2. `Push`/`pop` go through `container/heap`; `highestPriority()` returns
   `pq.ps[0].priority()` and `0` on empty. Inferable: partially.
3. `Take(n)` returns nil for `n <= 0`; pops `n` entries in priority order
   when `n < Len`. Inferable: doc.
4. `Take(n >= Len)` returns the internal heap array verbatim (after
   zeroing the slice) — order is HEAP-ARRAY order, not sorted priority
   order. Push 1..6, `Take(1)` -> `[6]`, then `Take(10)` ->
   `[5 4 2 1 3]`. Inferable: no — the unsorted bulk path is an
   implementation leak.
5. `clean()` removes canceled entries via `heap.Remove` in place;
   `reset()` nils every slot then truncates; `all()` copies the internal
   array (heap order, unsorted). Inferable: partially.
6. `Take` leaves `pq.ps` empty when draining everything (`Len()` becomes 0,
   `highestPriority()` 0). Inferable: yes.
