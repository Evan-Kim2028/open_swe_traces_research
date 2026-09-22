# Bug report — leafmsg

**Title:** Leaf-node messages with queue groups lose the queue list;
routed-leaf sub keys collide with plain routed sub keys.

**Symptoms:**
- A leaf `LMSG foo + reply q1 q2 42` is silently misparsed: the queue
  group names (`q1`, `q2`) never reach `c.pa.queues`, so downstream
  queue-group delivery treats every queue member as a candidate and
  delivers duplicates.
- `LHMSG` ignores the header-length field, so `pa.hdr` stays zero and
  the payload is treated as body-only; messages carrying NATS headers
  arrive garbled.
- Routed-sub bookkeeping keys omit the kind prefix and origin, so a
  leaf-origin subscription `"L foo q1 leafX"` collides with the plain
  routed sub `"R foo q1"` in `acc.rm`, producing phantom interest or
  lost interest across routes.

**Reproduction:** send a queue-group `LMSG` through a leaf connection
and observe `pa.queues` empty; or create a routed sub and a leaf-origin
sub on the same subject+queue and observe one `rm` key.

**Expected:** queues populated per the `+`/`|` indicator protocol,
`hdr`/`hdb` honoured for `LHMSG`, and keys carrying the `R`/`N`/`L`
prefix plus origin suffix.
