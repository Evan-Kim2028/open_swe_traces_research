# Details — sigblock

1. A signature block is recognised ONLY at a line boundary — a `-----BEGIN …-----` marker
   mid-line does not count; the scan advances whole lines. Inferable: partially.
2. `-----BEGIN PGP MESSAGE-----` counts as a signature start alongside the SIGNATURE
   markers — a PGP message block is treated as signature material. Inferable: partially —
   the format table is visible but which entries participate is the excised mapping.
3. The split position is the LAST block's start, not the first — text before it is
   message, everything from it on is signature, so a marker inside prose followed by more
   text still peels if a later line also starts a block. Inferable: doc — the comment
   says "the position of the last one".
4. A payload containing more than one signature-start line counts as multi-signature —
   the count feeds rejection of double-signed objects. Inferable: doc — the comment ties
   it to upstream's check.
5. Signature headers match only the canonical spellings `gpgsig ` and `gpgsig-sha256 `
   WITH the trailing space — `gpgsig` alone, `gpgsigfoo`, or other gpgsig-prefixed extra
   headers pass through. Inferable: doc — the comment says exactly this.
6. Stripping drops the signature header AND every following line that begins with a space
   (continuations); the first non-space line ends the continuation run — but a SECOND
   signature header directly after also gets dropped (the skip state chains). Inferable:
   no.
7. Once the blank line closing the header block is seen, everything after is copied
   verbatim — a `gpgsig`-looking line inside the BODY is never stripped. Inferable:
   partially.
8. For tag objects the trailing inline signature is truncated from the raw bytes BEFORE
   header stripping; for commits it is not — the two object types differ in exactly this
   step. Inferable: partially.
9. A final line without a trailing newline is still processed and copied — end-of-input
   mid-line is not an error. Inferable: no.
10. The strip writes into the destination object with its type set — destination type is
    assigned from the requested object type, not copied from source. Inferable: partially.
