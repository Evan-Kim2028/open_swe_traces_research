# Details — fsnode

1. A file's node hash is blob-hash bytes followed by file-mode bytes —
   content and mode both participate in the comparison. Inferable:
   doc — Hash's comment states the concatenation.
2. A directory's hash is always 24 zero bytes — directories never
   compare by content. Inferable: doc.
3. When an index is supplied and the entry's size, mode and mtime all
   match, the stored index hash is reused instead of re-reading the
   file — metadata is authoritative only when the racy-git check also
   passes. Inferable: doc — the Options comment describes the
   optimization.
4. Racy guard: a file whose mtime is equal to or newer than the index's
   own ModTime is always content-hashed, even if other metadata matches.
   Inferable: partially — the doc comment names the condition, the
   `!Before` comparison is a detail.
5. If no index ModTime is available (in-memory index), metadata alone is
   never trusted — always fall back to content hashing. Inferable: no.
6. With AutoCRLF, a text file's hash is computed over its LF-normalized
   content and the declared size shrinks by the CRLF count; binary files
   are hashed raw. Inferable: partially.
7. A symlink's hash is over its target path bytes — the link is never
   followed. Inferable: partially.
8. Submodule paths report the recorded submodule commit hash with
   submodule mode — the directory is not descended. Inferable: partially.
9. `.git` is always excluded from children; sockets are skipped; a
   directory that disappears between walk and listing yields no
   children rather than an error. Inferable: partially.
10. Ignored entries are skipped only when the IgnoreScope matches AND
    the path (or, for directories, any descendant) is absent from the
    index — tracked files inside ignored directories still walk.
    Inferable: doc — the Options comment states the tracked-entry rule.
11. Each directory's ignore scope is derived lazily from its own listing
    — a .gitignore is read only in visited directories, never below an
    excluded one. Inferable: partially.
12. Hash is computed lazily on first Hash() call and cached — later
    filesystem changes are invisible. Inferable: doc.
