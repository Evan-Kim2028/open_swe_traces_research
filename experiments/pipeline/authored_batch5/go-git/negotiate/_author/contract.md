# Contract — negotiate

`plumbing/transport.NegotiatePack` and its internal helpers — the
fetch-negotiation state machine: have-batch windowing, ACK processing,
stateless-RPC requeueing, capability selection, shallow handling, and
object-format reconciliation. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Window growth.** The have-batch window starts small, doubles each
   round up to a 16384 ceiling, then grows by a fixed increment under a
   persistent pipe and by a slower multiplicative curve under stateless
   RPC. Asserted on the batching helper's boundary behaviour — the
   ceiling and the two growth regimes, not the exact curve.
   Covered by `TestDetail01`.
2. **Vein budget (shape).** After a continue, an ACK-less stretch ships
   at most a bounded number of haves regardless of how many remain; the
   budget is restored only by new server ACKs. Asserted as a bound on
   haves-per-stretch, not a literal counter. Covered by `TestDetail02`.
3. **Termination.** Negotiation stops when haves are exhausted and emits
   a trailing `done` round once the server reports ready. Covered by
   `TestDetail03`.
4. **No-change fast path.** When wants are a subset of haves and no
   shallow state exists, no `want` line is written; the writer is
   flushed and closed and the call reports no-change. Covered by
   `TestDetail04`.
5. **Stateless rounds.** Under stateless RPC the writer is closed after
   each round, and the next round re-sends the full request preamble
   plus every accumulated common have. Covered by `TestDetail05`.
6. **ACK state transitions (shape).** Continue and ready reset the vein
   and set their flags; common accumulates the common set and implies
   continue. Under stateless RPC a common ACK also appends the hash to
   the requeue list and resets the vein — but only for a hash not
   already common; a duplicate changes nothing. Covered by
   `TestDetail06`.
7. **multi_ack preference.** When the server offers both ack modes the
   request asks for `multi_ack_detailed` and never names both.
   Covered by `TestDetail07`.
8. **Progress channel.** With a progress channel set and both sideband
   modes offered, the request asks for `side-band-64k` and not plain
   `side-band`; with no progress channel the request carries no progress
   capability token at all. Covered by `TestDetail08`.
9. **Depth requires shallow.** `Depth>0` without the shallow capability
   hard-fails; with it, the request announces the repository's current
   shallow list. Asserted on the requirement and the presence of the
   announced hashes, not the full line layout. Covered by
   `TestDetail09`.
10. **Format reconciliation.** An object-format mismatch is a hard
    error, except the fresh-clone case — unset client format, sha256
    server, placeholder HEAD — which adopts the server algorithm through
    the format setter. Covered by `TestDetail10`.
11. **Absent server format (shape).** A server advertising no
    object-format is treated as sha1: a non-sha1 client fails with an
    error naming the format rather than corrupting the pack. Covered by
    `TestDetail11`.
12. **Shallow-update decode.** With depth requested, the response opens
    with a shallow-update section that is consumed before the ACK lines
    and surfaced through the returned `ShallowUpdate`; without depth the
    section is not consumed. Covered by `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | no — boundaries and regimes, not the curve |
| TestDetail02 | 2 | no — bound only |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | no — observable state transitions |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | no — error shape naming the format |
| TestDetail12 | 12 | partially |
