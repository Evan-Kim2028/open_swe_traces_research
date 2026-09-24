# Details — renamedet

1. Detection runs only when BOTH additions and deletions exist — a changeset
   missing either side is returned without scoring. Inferable: partially.
2. Exact renames match on identical content hash AND identical file mode —
   same hash with a mode change stays add+delete, not a rename. Inferable:
   partially — the same-mode gate is an upstream rule.
3. With multiple deletes sharing the added file's hash, the winner is the
   most path-similar name — the losers revert to plain deletions. Inferable:
   partially — bestNameMatch is named in the detector comment.
4. With multiple adds per hash, a bounded similarity matrix over name scores
   picks pairs greedily from highest score, each side used once — the
   matrix, not iteration order, decides. Inferable: partially.
5. Name similarity blends directory prefix+suffix scores (25% each) with
   filename-suffix score (50%) — weights are fixed fractions, not equal.
   Inferable: no — the weighting is an upstream constant.
6. Content renames are skipped entirely when OnlyExactRenames is set, and
   abandoned wholesale when the larger side exceeds RenameLimit — the limit
   refuses, it does not truncate. Inferable: partially — the doc comment
   covers only-exact, the limit policy is the detail.
7. Non-regular files (links, submodules, executables) are excluded from
   content pairing — only regular files enter the matrix. Inferable: no.
8. File sizes are inflated by one before the size gate so empty files can
   still score — min*100/max must reach RenameScore to stay a candidate.
   Inferable: no — the +1 dodge is arbitrary.
9. A pair's score is content similarity weighted 99 to name similarity
   weighted 1, both normalised to the same range — content dominates.
   Inferable: no.
10. The similarity index hashes line regions for text and fixed 64-byte
    blocks for binary, folding CRLF to LF in text — a lone CR without LF
    still counts. Inferable: partially — the index doc comment describes
    regions, the CR handling is the detail.
11. The index starts at 256 slots and doubles until the 30-bit ceiling,
    packing 32-bit key + 32-bit count into one word — growth past the cap
    reports index-full, not panic. Inferable: doc — the 1MiB bound and
    growth policy are in the type comment.
12. Pair scores equal common region count over the larger region count
    times the scale — symmetrical, and two empty files score the maximum.
    Inferable: doc — the score comment defines the fraction and the
    degenerate case.
13. The result concatenates remaining adds, remaining deletes, then matched
    modifies — and is stable-sorted as one list at the end, not per bucket.
    Inferable: partially — the ordering is observable, the bucket order is
    the detail.
14. A nil options pointer uses the package defaults — detection is never
    skipped for nil opts. Inferable: doc — the entry comment says so.
