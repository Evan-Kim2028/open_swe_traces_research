# Details — negotiate

1. Haves ship in growing batches — the window doubles each round (up to
   16384), then grows 10% per round under stateless RPC or by a fixed 32
   under a persistent pipe. Inferable: no — the constant names survive
   but the growth curves are detail.
2. After the server says "continue", at most 256 haves are sent per
   ACK-less stretch — the vein budget resets only on new ACKs.
   Inferable: no.
3. The loop ends when haves run out, the vein budget is exhausted under
   multi-ack, or a trailing "done" round after the server reports ready.
   Inferable: partially.
4. If wants ⊆ haves and no shallow state, no request is sent at all —
   a flush + close + ErrNoChange instead. Inferable: partially.
5. In stateless mode the writer closes after each round and every
   round re-sends the accumulated common haves plus the full request
   preamble. Inferable: partially.
6. ACKContinue/ACKReady/ACKCommon each reset the vein counter; only
   ACKCommon accumulates into the common set, and only a *new* common
   hash resets the vein under stateless RPC. Inferable: no.
7. `multi_ack_detailed` is requested in preference to plain `multi_ack`
   — never both. Inferable: partially.
8. Progress chooses sideband-64k over sideband when both are offered;
   no progress channel selects no-progress. Inferable: partially.
9. Depth>0 requires the shallow capability or hard-fails; the request
   carries the repo's current shallow list. Inferable: partially.
10. Object-format mismatch is a hard error — except a fresh clone (unset
    client format + sha256 server + placeholder HEAD) which adopts the
    server's algorithm via the ObjectFormatSetter. Inferable: partially.
11. A server that advertises no object-format speaks sha1 only — a client
    repo on another algorithm fails fast rather than corrupting the pack.
    Inferable: no.
12. The shallow-update response is decoded once on the first round (or
    every round under stateless RPC) and only when depth was requested.
    Inferable: partially.
