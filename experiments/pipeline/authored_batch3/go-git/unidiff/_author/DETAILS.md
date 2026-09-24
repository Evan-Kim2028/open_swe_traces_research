# Details — unidiff

1. Each changed file starts `diff --git a/<from> b/<to>`; a mode change adds `old mode`/
   `new mode` in OCTAL; a path change adds `rename from`/`rename to`; a content change adds
   `index <h>..<h>` — with the mode suffix only when the mode did NOT change. Inferable:
   partially.
2. New files emit `new file mode`, `index 0000..hash`, `--- /dev/null` + `+++ b/<to>`;
   deletions mirror with `deleted file mode` and `+++ /dev/null`. The new-file `diff --git`
   line uses the DESTINATION path under both prefixes. Inferable: partially.
3. When only metadata changed (same hash, e.g. pure mode change or rename), NO index line,
   NO `---`/`+++` lines and no hunks appear — the header stands alone. Inferable: no.
4. A binary file patch replaces the `---`/`+++` pair with a single `Binary files X and Y
   differ` line, still after the index line. Inferable: partially.
5. The hunk header is `@@ -<line>[,<count>] +<line>[,<count>] @@` — the `,count` part is
   OMITTED exactly when the count is 1, and present (`,0`) for an empty side. Inferable:
   partially.
6. The line of context that got trimmed immediately before a hunk reappears after the `@@`
   as the section/function heading — with its newline stripped and a space before it.
   Inferable: no.
7. An unchanged run between two changes of at most twice the context length keeps both
   changes in ONE hunk; a longer run splits them, leaving exactly the context amount on each
   side and the remainder as leading context of the next hunk. Inferable: partially.
8. With context length zero the merging rule still applies — adjacent changes share a hunk
   and no context is ever emitted. Inferable: no.
9. A content line lacking a trailing newline is emitted followed by the
   `\ No newline at end of file` marker on its own line. Inferable: partially.
10. A non-empty patch message is written before all file patches, with a newline appended if
    it lacks one. Inferable: partially.
11. Content is split keeping each line's newline; a string ending in newline does NOT yield
    a phantom empty final line. Inferable: doc — the splitter's comment states it.
12. Color spans wrap the header block (Meta), the hunk header (Frag), the section heading
    (Func), and each content line by its operation — reset after every span. Inferable:
    partially — the key map is visible, the spans are not.
