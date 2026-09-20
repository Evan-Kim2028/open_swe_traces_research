# Details — inforefs

NOTE: this unit's doc comment is unusually complete — most commitments are spelled out in the
kept `Decode` doc comment, so `Inferable: doc` dominates. That is intentional: the batch needs a
control for "documented edge, still missed".

1. Every non-empty line must be `<hex>` TAB `<name>` — a line with no tab fails the WHOLE body
   with the invalid-info/refs error. Inferable: doc.
2. The hash field must be exactly 40 or 64 hex digits AND valid hex; any other length or
   character fails the whole body — including refs already read before the bad line.
   Inferable: doc.
3. A rejected body leaves the receiver's reference list UNCHANGED — partial results are not
   appended. Inferable: doc.
4. Empty lines are skipped silently; they do not count as malformed. Inferable: doc.
5. A name is validated after removing ONE optional `^{}` suffix; a name that fails is SKIPPED —
   the line is dropped but the body still decodes. Contrast with the hash rule, which is fatal.
   Inferable: doc.
6. The `^{}` suffix is preserved in the decoded name — a peeled ref lands named `base^{}` even
   though only `base` was validated. Inferable: doc.
7. A line carrying a valid hash and tab but NO name at all is one of the skipped lines (the
   empty string is not a usable name). Inferable: partially.
8. A line longer than the scanner's token limit is a malformed-body failure, not a passthrough
   scanner error; a genuine read failure IS passed through unchanged. Inferable: doc.
9. Lines are scanned with one trailing carriage return tolerated — a name holding a second CR
   fails the name rule and the line is skipped. Inferable: partially.
10. Encode writes `<hash>\t<name>\n` per reference in stored order — no header, no terminator.
    Inferable: yes.
11. The hash field is length-checked before hex parsing because the parser would silently pad a
    short hex string rather than reject it. Inferable: doc.
