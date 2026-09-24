# Details — srvresp

1. Valid response lines are `ACK <hash>`, `ACK <hash> <status>`, or `NAK` — any other content is
   an "unexpected content" failure. Inferable: partially.
2. A `NAK` line ends the response cleanly — decoding stops and succeeds; NAK is not an error.
   Inferable: no — NAK-as-terminator vs NAK-as-failure is a guess.
3. An `ACK` line with a status field keeps decoding; an `ACK` line WITHOUT a status (plain
   `ACK <hash>`, the non-multi_ack form) is treated as the LAST line — decoding stops after it.
   Inferable: no.
4. Status words map `continue`/`common`/`ready` onto the three status constants; an unrecognised
   status word leaves the status at its zero value and does NOT fail the decode.
   Inferable: no.
5. The zero status value prints as the empty string (only the three named constants have text).
   Inferable: doc — the constants are kept in the source, but the empty-string mapping is a
   fall-through detail.
6. An `ACK` line shorter than the minimum for `ACK <40-hex>` — or one missing the hash field —
   is malformed; the hash keeps any needed trailing-newline trimming. Inferable: partially.
7. Encoding an empty response emits a single `NAK` line — not a flush, not nothing.
   Inferable: no.
8. Encoding with status-bearing acks emits `ACK <hash> <status>` per ack; with status-less acks
   it emits a single `ACK <hash>` for the first ack and stops — remaining acks are not written.
   Inferable: no.
9. Every valid ACK line appends to the ack list — including ones whose status word was
   unrecognised (they land with a zero status, not dropped). Inferable: partially.
10. A blank/flush line inside the response is an error, not a terminator — only NAK or a bare
    `ACK <hash>` may end it. Inferable: partially.
