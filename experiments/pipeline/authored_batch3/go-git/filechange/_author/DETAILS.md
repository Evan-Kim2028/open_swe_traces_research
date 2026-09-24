# Details — filechange

1. A change with both sides empty is malformed; exactly-one-side-empty maps to insert or
   delete; both present maps to modify. Inferable: doc — the method comment names the
   three actions.
2. "Empty" is full-struct equality against a zero `ChangeEntry` — an entry carrying only a
   name (no tree) still counts as PRESENT, so a half-populated side is treated as real.
   Inferable: no.
3. `Files` resolves the entry to a real file only for the sides the action needs — insert
   yields a nil `from`, delete a nil `to` — and a non-file entry (dir/submodule mode)
   silently yields `(nil, nil, nil)` with NO error. Inferable: no.
4. The display name of a change prefers the FROM path — a delete reports the old name; a
   rename-like modify reports the old path. Inferable: no.
5. `Changes.Less` sorts on that same preferred name — deletions sort by their OLD path,
   not by the new one. Inferable: partially.
6. `File.Lines` splits on `\n` only — carriage returns are NOT stripped, so CRLF content
   yields `\r`-terminated lines despite the doc comment claiming end-of-line characters
   are stripped. Inferable: no.
7. A content string ending in a newline yields no phantom empty tail — but an empty file
   yields an EMPTY slice, not a one-empty-string slice. Inferable: partially.
8. Binary detection is content-based via the shared helper over the blob stream — a file
   named `.txt` with NUL bytes still reports binary. Inferable: partially.
9. `Change.String` renders `<Action: X, Path: Y>` and a malformed change renders the
   literal `malformed change` rather than failing. Inferable: partially.
10. `Contents` reads the whole blob into memory — the string IS the content, not a
    truncated preview. Inferable: yes.
