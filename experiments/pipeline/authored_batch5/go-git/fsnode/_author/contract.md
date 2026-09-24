# Contract — fsnode

`NewRootNode`/`NewRootNodeWithOptions` adapt a billy filesystem to
`noder.Noder`: hash layout, index-metadata fast path, AutoCRLF
normalization, symlink and submodule handling, walk exclusions, and
scoped ignore rules. Every commitment below is covered by a hidden test;
every hidden test maps to a commitment.

## Commitments

1. **File hash layout.** A file's node hash is the blob-hash bytes of its
   content followed by file-mode bytes — 24 bytes total — and changing
   either content or mode changes the hash. Covered by `TestDetail01`.
2. **Directory hash.** A directory's hash is always 24 zero bytes.
   Covered by `TestDetail02`.
3. **Metadata fast path.** When an index entry's size, mode and mtime all
   match the file and the entry is not racy, the stored index hash is
   reused — observable because a stored hash that differs from the
   content hash is reported verbatim. Covered by `TestDetail03`.
4. **Racy guard.** A file whose mtime equals or exceeds the index
   `ModTime` is content-hashed even when other metadata matches.
   Asserted at both the equal and newer boundary. Covered by
   `TestDetail04`.
5. **Missing index ModTime (shape).** When the index carries no
   `ModTime`, metadata alone is never trusted — the asserted shape is
   that the stored index hash is never adopted and the produced hash is
   the content hash. Covered by `TestDetail05`.
6. **AutoCRLF.** With `AutoCRLF`, a text file's hash equals the blob hash
   of its LF-normalized content; a binary file's hash equals the blob
   hash of its raw bytes. Covered by `TestDetail06`.
7. **Symlink target.** A symlink's hash is the blob hash of its target
   path bytes — the link is never followed to its target's content.
   Covered by `TestDetail07`.
8. **Submodule hash.** A path recorded in the submodules map reports the
   recorded commit hash as its hash prefix; distinct recorded commits
   yield distinct node hashes. Mode suffix encoding is not pinned.
   Covered by `TestDetail08`.
9. **Walk exclusions.** `.git` never appears among children; socket
   entries are skipped; a directory that disappears between the parent's
   listing and its own listing yields no children and no error. Covered
   by `TestDetail09`.
10. **Ignore scope vs index.** An ignored entry is skipped only when the
    scope matches and neither it nor (for directories) any descendant is
    in the index — a tracked file matching an ignore rule still walks,
    and an ignored directory containing a tracked file is still entered.
    Covered by `TestDetail10`.
11. **Lazy scope derivation.** A `.gitignore` in a visited directory
    applies to that directory's children, and an excluded directory's
    `.gitignore` is never read — asserted via a `!` negation inside an
    excluded directory that must not resurrect it. Covered by
    `TestDetail11`.
12. **Lazy cached hash.** `Hash()` computes once and caches — modifying
    the file afterwards does not change an already-queried node's hash,
    while a freshly constructed node sees the new content. Covered by
    `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | no — shape only |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | doc |
