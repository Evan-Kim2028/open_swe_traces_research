# Details — objpatch

1. An empty change list still produces a Patch — carrying the message and
   zero file patches, not an error. Inferable: partially.
2. Submodule (gitlink) entries never read blob content — the diff body is
   the synthetic `Subproject commit <hash>` line on whichever side is a
   submodule. Inferable: doc — the comment spells the format.
3. A change where either side is a submodule is diffed as text over those
   synthetic lines — added or removed subprojects diff against empty
   content. Inferable: partially.
4. Binary content suppresses chunks entirely — the file patch still names
   both sides but carries no diff body. Inferable: partially.
5. Every per-change and per-chunk loop honours context cancellation and
   surfaces the same ErrCanceled — checked between items, not inside
   reads. Inferable: partially — ErrCanceled is kept, the check cadence
   is the detail.
6. Diff operation mapping is one-to-one with the dmp triple — equal,
   delete, insert — in that order. Inferable: doc — the fdiff.Operation
   names are visible.
7. A change entry is "empty" (no file side) when its mode is not a file —
   EXCEPT submodules, which always count as present so their gitlink is
   diffed. Inferable: doc — the comment states it.
8. FileStats ignore patches with no chunks — binary files and submodule
   bumps produce no stat row at all. Inferable: partially.
9. Stat names pick the surviving path for add/delete, and `from => to`
   for a rename — the arrow with spaces is fixed. Inferable: partially —
   the join shape is observable output, the spacing arbitrary.
10. Additions and deletions count newline characters, plus one when the
    chunk doesn't end in a newline — an unterminated last line counts.
    Inferable: no.
11. printStat scales the +/- graph linearly only when a row's total
    exceeds the width cap — the bar is `1 + it*(width-1)/max`, so nonzero
    counts always show at least one mark. Inferable: doc — the upstream
    diff.c link is kept.
12. Stats rows align name and change-count columns to the widest entry,
    with the graph starting after a ` | ` separator. Inferable: no —
    print layout is an arbitrary choice.
13. Patch.String on encode failure returns a `malformed patch:` string
    rather than propagating the error. Inferable: no.
14. Patch.Encode always uses the default three context lines through the
    unified encoder. Inferable: partially — DefaultContextLines is a
    visible const.
