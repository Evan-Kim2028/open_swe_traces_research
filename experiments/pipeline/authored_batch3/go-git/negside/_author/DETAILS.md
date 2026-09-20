# Details — negside

1. A shallow update is a stream of data lines `shallow <hash>` / `unshallow <hash>` ending in a
   flush; each kind lands in its own list in arrival order. Inferable: doc.
2. A shallow-update stream cut off by end-of-input BEFORE the flush is accepted silently — the
   scanner reports no error on plain EOF — while a push-option stream missing its flush is a
   truncated-input failure. The two decoders disagree deliberately. Inferable: no.
3. The update hash is taken from the tail of a fixed-length line: `shallow ` lines must be
   exactly prefix+40 chars, `unshallow ` lines prefix+2+40 — any other length is malformed, and
   the hash digits themselves are NOT validated (non-hex is accepted). Inferable: no.
4. Lines are whitespace-trimmed before their keyword is matched, so leading/trailing space is
   tolerated; a line matching neither keyword is a malformed-line failure. Inferable: no.
5. Encoding writes all shallows first, then all unshallows, then a flush — regardless of the
   order they were decoded in. Inferable: partially.
6. Each push option is written as its own pkt-line payload with NO trailing newline — the
   pkt-line framing itself delimits options. Inferable: no.
7. Encoding validates EVERY option before writing any byte — a bad option in the middle of the
   list means nothing reaches the wire at all. Inferable: partially.
8. An option containing a non-graphic character (per unicode.IsGraphic — so space IS allowed,
   tab/newline/control are not) or exceeding the max payload size is an invalid-option failure.
   Inferable: partially.
9. Decode initialises the option list to a non-nil empty slice even for an empty stream.
   Inferable: no.
10. A push-option line containing non-graphic bytes fails on decode too — same predicate both
    directions. Inferable: partially.
