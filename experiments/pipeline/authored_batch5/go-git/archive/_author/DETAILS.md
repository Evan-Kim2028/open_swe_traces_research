# Details — archive

1. Without allowUnreachable, raw hashes are rejected with ErrOnlyRefNames
   and relative expressions (`^~@{}`) with ErrRelativeExpressions —
   only ref names and ref:path are allowed. Inferable: doc — the
   ResolveTreeish comment states the policy.
2. `ref:path` strips at the FIRST colon — a path may itself contain
   colons; the sub-path must resolve to a directory entry, not a file.
   Inferable: partially.
3. Ref resolution tries the bare name, then `refs/heads/`, then
   `refs/tags/` in that order — a branch name beats a tag of the same
   name. Inferable: partially.
4. Annotated tags unwrap to their target repeatedly until a non-tag
   object is reached — chains of tags work. Inferable: partially.
5. Tree-ish resolution timestamps differ by input kind: commits and
   tags use the committer time, but a bare tree uses the current time.
   Inferable: doc — the comment cites git-archive semantics.
6. Tar output carries a PAX global header named `pax_global_header`
   whose comment is the commit hash — `git get-tar-commit-id` reads it
   back. Inferable: doc.
7. Directories in tar get mode `mode|0777` minus umask; regular files get
   `mode|0666` (or `|0777` if executable) minus umask; symlinks are
   always 0777 and carry the link target as Linkname with size 0.
   Inferable: partially.
8. A prefix ending in `/` emits an explicit directory entry before the
   files; the prefix is prepended to every member name. Inferable:
   partially.
9. A symlink target larger than 64KiB is a hard error — the tar format
   cannot carry it. Inferable: no — the constant is kept but the bound
   is a policy choice.
10. Path filters match exact names, `name/` prefixes, parent dirs of
    `name`, and glob patterns — and if any filter list is supplied but
    nothing matched, the write fails with ErrPathspecNoMatch. Inferable:
    doc — the supported-pattern list is in the comment.
11. Prefixes containing `..` path segments or a leading `/` or `\` are
    rejected before any output is written. Inferable: doc.
12. Zip output records the commit hash as the archive comment, uses
    Deflate, and maps symlink mode to the 0o120000 bit — directories are
    not emitted as members. Inferable: partially.
