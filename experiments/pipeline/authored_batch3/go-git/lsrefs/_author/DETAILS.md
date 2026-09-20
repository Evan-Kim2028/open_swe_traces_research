# Details — lsrefs

1. Argument encode order is fixed: `peel`, then `symrefs`, then `unborn`, then one
   `ref-prefix <p>` line per prefix — each argument its own packet line, nothing when the flag
   is unset. Inferable: partially.
2. Arguments are validated BEFORE anything is written — a bad ref-prefix fails the whole encode
   and leaves the stream empty. Inferable: no.
3. A ref-prefix is rejected when empty or when it contains whitespace or control characters
   (incl. NUL and non-ASCII space/control). Inferable: partially.
4. Argument decode ignores lines it does not recognise and skips empty lines; a flush or
   end-of-input terminates. Inferable: partially.
5. When ref-prefix arguments reach the kept bound (65536) the accumulated list is DROPPED
   entirely and further prefixes ignored — matching upstream's "advertise everything" fallback.
   Inferable: no — the constant is visible but the drop-not-cap behaviour is not.
6. Response line grammar: `<oid> SP <refname> [SP symref-target:<t>] [SP peeled:<oid>]`, or
   `unborn SP <refname> SP symref-target:<t>` for an unborn HEAD. Inferable: doc — the type doc
   comment states the grammar and stays in the tree.
7. On encode, a `^{}` peeled entry never gets its own line — it folds into its base ref's
   `peeled:` attribute. Inferable: partially.
8. On encode a symbolic ref emits `symref-target:<target>` plus the target's resolved oid when
   that target's hash is present in the same list and non-zero — otherwise the literal `unborn`
   takes the oid position. Inferable: no.
9. On decode an `unborn` line REQUIRES a `symref-target:` attribute — `unborn <ref>` alone is
   malformed. Inferable: no.
10. A `peeled:` attribute on decode produces a SECOND reference named `<base>^{}` appended right
    after the base entry. Inferable: partially.
11. A line with `symref-target:` decodes to a SYMBOLIC reference even when a full object id is
    present — the oid is discarded in favour of the symref. Inferable: no.
12. Object ids are parsed strictly — exactly 40 or 64 hex — rather than through the padding
    parser used elsewhere in the tree. Inferable: partially.
13. Unknown attributes on a ref line are ignored, not errors; fields split on runs of spaces,
    so repeated spaces don't produce empty tokens. Inferable: partially.
