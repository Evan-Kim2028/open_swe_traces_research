# Details — tagparse

1. The first three headers must be `object`, `type`, `tag` in exactly that order — a wrong
   order or a missing one is malformed, and `tag` is the NAME not the target. Inferable:
   doc — the state comments mirror upstream's strict parse.
2. `tagger` is optional at its canonical position; a tag without it parses with a zero
   signature and encodes WITHOUT the tagger line. Inferable: partially.
3. A duplicate of a canonical header out of position (a second `object` or `tag` in the
   header block) is silently DROPPED — not an error, not an override. Inferable: doc — the
   dispatch comment states it.
4. Unknown headers are silently dropped — they are neither stored nor rejected. Inferable:
   doc.
5. The `gpgsig-sha256` header folds multi-line values: each continuation line strips exactly
   ONE leading space, and repeated occurrences of the header CONCATENATE into the stored
   signature. Inferable: doc — the continuation-state comment cites upstream.
6. Everything after the blank line is the message, except that a trailing inline PGP
   signature block is peeled into `Signature` — the peel happens on the LAST signature
   marker found, not the first. Inferable: partially — the kept `parseSignedBytes` helper
   reveals last-match wins.
7. Encode emits headers in canonical order and puts `gpgsig-sha256` between the tagger line
   and the blank separator — its value is re-folded with single-space continuations.
   Inferable: doc — the insertion-point comment cites upstream.
8. `Signature` is appended after the message VERBATIM with no separator inserted — a message
   not ending in newline corrupts the object (the warning comment stays). Inferable: doc.
9. A signature counts as "zero" only when name AND email are empty AND the timestamp is the
   zero time — a name-only tagger still emits the line. Inferable: no.
10. `EncodeWithoutSignature` has two paths: fields unchanged since decode → the ORIGINAL raw
    bytes are streamed with signature material stripped, preserving exact signed bytes; any
    payload-field mutation → re-encode from struct fields. Inferable: doc — the method
    comment describes both.
11. The source-match comparison deliberately EXCLUDES `Signature` and `SignatureSHA256` —
    mutating them still yields the raw-bytes path. Inferable: doc — the comment calls it
    intentional.
12. A pushed-back line is re-served exactly once — the scanner's one-deep pushback means a
    state can decline a line without losing it. Inferable: no.
