# Contract — renamedet

`DetectRenames` regroups add/delete change pairs into modify entries when
the added file matches a deleted one — exactly by hash+mode, or by a
content-similarity matrix bounded by score and limit options. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Both sides required.** Detection runs only when additions AND deletions
   both exist; a changeset missing either side is returned without scoring.
   Covered by `TestDetail01`.
2. **Exact = hash + mode.** Exact renames require identical content hash and
   identical file mode; same hash with a mode change stays add+delete.
   Covered by `TestDetail02`.
3. **Best name wins.** With multiple deletes sharing the added file's hash,
   the most path-similar name wins; losers revert to plain deletions.
   Covered by `TestDetail03`.
4. **Greedy matrix pairing.** With multiple adds per hash, a similarity
   matrix over name scores pairs greedily from highest score, each side
   used once. Covered by `TestDetail04`.
5. **Fixed name weights (shape).** Name similarity is a fixed blend of
   directory prefix, directory suffix, and filename suffix scores — the
   asserted shape is bounded, monotone-in-similarity scoring, not the
   weight literals. Covered by `TestDetail05`.
6. **Exact-only and limit.** `OnlyExactRenames` skips content pairing
   entirely; exceeding `RenameLimit` abandons content detection wholesale —
   the limit refuses, it does not truncate. Covered by `TestDetail06`.
7. **Regular files only (shape).** Non-regular files are excluded from
   content pairing; a symlink-mode change is never paired. Covered by
   `TestDetail07`.
8. **Empty files participate (shape).** Sizes are inflated before the size
   gate so empty files can still score — asserted shape is that an empty
   file flows through detection without error or silent loss. Covered by
   `TestDetail08`.
9. **Content dominates (shape).** Pair scores weight content similarity far
   above name similarity — asserted shape is that content-similar files
   with dissimilar names still pair. Covered by `TestDetail09`.
10. **Region hashing.** The similarity index hashes line regions for text
    and fixed-size blocks for binary, folding CRLF to LF; a lone CR still
    counts as a boundary. Covered by `TestDetail10`.
11. **Bounded index.** The index starts at 256 slots and doubles up to a
    bounded ceiling, packing key and count into one word; growth past the
    cap reports index-full rather than panicking. Covered by
    `TestDetail11`.
12. **Score definition.** Pair score is common region count over the larger
    region count times the scale — symmetrical, with two empty files
    scoring the maximum. Covered by `TestDetail12`.
13. **Whole-list sort.** The result concatenates remaining adds, remaining
    deletes, then matched modifies, stable-sorted as one list at the end.
    Covered by `TestDetail13`.
14. **Nil options.** A nil options pointer uses package defaults; detection
    is never skipped for nil opts. Covered by `TestDetail14`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | no — shape only |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | no — shape only |
| TestDetail08 | 8 | no — shape only |
| TestDetail09 | 9 | no — shape only |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | doc |
| TestDetail12 | 12 | doc |
| TestDetail13 | 13 | partially |
| TestDetail14 | 14 | doc |
