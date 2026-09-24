# Contract — objpatch

`plumbing/object` patch construction — `getPatch`/`getPatchContext`,
submodule (gitlink) diffing, chunk mapping, `FileStats`, and the
`printStat` layout. Every commitment below is covered by a hidden test;
every hidden test maps to a commitment.

## Commitments

1. **Empty patch is still a Patch.** A call with no changes returns a
   `Patch` carrying the message and zero file patches — never an error.
   Covered by `TestDetail01`.
2. **Synthetic submodule content.** A gitlink entry's diff body is the
   single line `Subproject commit <hash>`; non-submodule and empty
   entries produce no synthetic content and are not classified as
   submodules. Covered by `TestDetail02`.
3. **Submodule diffs over synthetic lines.** Adding a submodule diffs
   its synthetic line against empty content (addition only); a bump
   deletes the old `Subproject commit` line and adds the new one.
   Covered by `TestDetail03`.
4. **Binary suppression.** A change touching binary content produces a
   file patch with no chunks that still names both sides and reports
   itself binary. Covered by `TestDetail04`.
5. **Cancellation.** A canceled context surfaces `ErrCanceled`, and the
   check runs per-change and per-chunk — a cancel landing inside the
   chunk loop of a single change still aborts with `ErrCanceled`.
   Covered by `TestDetail05`.
6. **Operation mapping.** Chunk operations map the diff engine's
   equal/delete/insert onto `Equal`/`Delete`/`Add` faithfully: the
   delete+equal chunks reassemble the from-content and the add+equal
   chunks the to-content, with each kind present for a modify change.
   Covered by `TestDetail06`.
7. **Empty-side rule.** A change entry counts as having a file side iff
   its mode is a file mode — or it is a submodule. Directory and
   absent entries are empty and expose neither path nor hash. Covered
   by `TestDetail07`.
8. **Chunkless patches make no stat row.** A binary change contributes
   no `FileStat` row while a text change in the same patch does.
   Covered by `TestDetail08`.
9. **Stat naming.** Add and delete rows name the surviving path; a
   rename row joins the two paths with the literal arrow ` => `.
   Covered by `TestDetail09`.
10. **Line counting.** Additions and deletions count newlines plus one
    for a chunk not ending in a newline — an unterminated last line
    still counts. Covered by `TestDetail10`.
11. **Graph scaling.** Rows under the width cap print marks equal to
    the literal counts; a row whose total exceeds the cap scales each
    nonzero component to at least one mark per the documented linear
    formula. Covered by `TestDetail11`.
12. **Column alignment (shape).** Every stats row carries a `|`
    separator at the same column, with the count and graph after it.
    Covered by `TestDetail12`.
13. **String/Encode equivalence (shape).** `Patch.String` yields exactly
    the bytes `Encode` produces for a well-formed patch. The
    `malformed patch:` error branch is unreachable through `String`
    because its internal buffer cannot fail — only the equivalence is
    asserted. Covered by `TestDetail13`.
14. **Three context lines.** `Encode` renders hunks with exactly three
    context lines through the unified encoder — a mid-file change in a
    nine-line file produces a hunk spanning lines 2–8 on both sides.
    Covered by `TestDetail14`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially — cancellation, not exact cadence |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | doc |
| TestDetail08 | 8 | partially — binary clause; submodule stat-row claim not asserted (reference behaviour emits a stat row for bumps since the synthetic diff has chunks) |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | no — observable counting rule |
| TestDetail11 | 11 | doc |
| TestDetail12 | 12 | no — alignment shape only |
| TestDetail13 | 13 | no — reachable half only |
| TestDetail14 | 14 | partially |
