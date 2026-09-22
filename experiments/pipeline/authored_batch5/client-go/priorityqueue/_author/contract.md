# Contract (L2) — priorityqueue

`NewPriorityQueue` returns a queue over `Item` (an interface with
`priority() uint64` and `isCanceled() bool`). The queue is ordered by
`priority()` with the largest value on top: `highestPriority()` reports the
top item's priority and `0` when the queue is empty, and `pop`/`Take(1)`
return that same top item. `Take(n)` returns nil for `n <= 0` without
touching the queue; for `0 < n < Len` it returns the `n` top-priority items
in pop order; for `n >= Len` it drains the queue and returns every queued
item (element order is internal). `clean()` removes canceled entries in
place and leaves live entries queued; `reset()` empties the queue entirely;
`all()` returns every queued item as a copy — mutating the result must not
corrupt the queue. After any full drain `Len()` is 0 and the queue stays
usable.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | the largest `priority()` sits on top: `highestPriority()` and `Take(1)` yield the maximum pushed priority |
| `TestDetail02` | `highestPriority()` is the top item's priority, `0` on empty, and agrees with what pops next |
| `TestDetail03` | `Take(n)` is nil for `n <= 0` and pops the `n` top-priority items in pop order when `n < Len` |
| `TestDetail04` | `Take(n >= Len)` drains the queue and returns every queued item |
| `TestDetail05` | `clean()` drops canceled entries only, `reset()` empties the queue, `all()` returns every queued item as a copy |
| `TestDetail06` | draining via `Take` leaves `Len() == 0` and `highestPriority() == 0` |
