# Contract — archive

`internal/archive` implements git-upload-archive: treeish resolution under a
ref-name allowlist, tar/zip emission with PAX/comment commit stamping, path
filters, and prefix safety. Every commitment below is covered by a hidden
test; every hidden test maps to a commitment.

## Commitments

1. **Ref allowlist.** Without `allowUnreachable`, raw hashes are rejected
   with `ErrOnlyRefNames` and relative expressions (`^`, `~`, `@{}`) with
   `ErrRelativeExpressions`; only ref names and `ref:path` are allowed.
   Covered by `TestDetail01`.
2. **First-colon split.** `ref:path` splits at the FIRST colon (paths may
   contain colons); the sub-path must resolve to a directory entry — a file
   produces a not-a-directory error naming the path. Covered by
   `TestDetail02`.
3. **Ref search order.** Bare names resolve through `refs/heads/` before
   `refs/tags/` — a branch beats a tag of the same name; fully-qualified
   refs still resolve. Covered by `TestDetail03`.
4. **Tag unwrapping.** Annotated tags unwrap repeatedly until a non-tag
   object; chains of tags resolve to the underlying commit's tree and hash.
   Covered by `TestDetail04`.
5. **Timestamps.** Commits and tags timestamp with the committer time; a
   bare tree resolves with the current time. Covered by `TestDetail05`.
6. **PAX commit stamp.** Tar output starts with a `pax_global_header` record
   whose comment is the commit hash; `GetTarCommitID` reads it back.
   Covered by `TestDetail06`.
7. **Tar modes.** Directories get mode|0777 minus umask, regular files
   mode|0666 minus umask (|0777 if executable); symlinks are 0777 with the
   link target in Linkname and size 0. Covered by `TestDetail07`.
8. **Prefix dir.** A prefix ending in `/` emits an explicit directory entry
   before files and is prepended to every member name. Covered by
   `TestDetail08`.
9. **Symlink bound (shape).** A symlink target over 64KiB is a hard error —
   asserted shape: `ErrSymlinkTargetTooLarge` is returned. Covered by
   `TestDetail09`.
10. **Path filters.** Filters match exact names, name-prefixes, slashed
    directory names, parents of filters, and glob patterns; a supplied
    filter list that matches nothing fails the write with
    `ErrPathspecNoMatch`. Covered by `TestDetail10`.
11. **Prefix safety.** Prefixes with `..` segments or leading `/` / `\` are
    rejected before any output is written — `HasInvalidPrefix` reports them
    and `WriteArchive` refuses. Covered by `TestDetail11`.
12. **Zip output.** Zip archives record the commit hash as the archive
    comment, use Deflate, emit symlink members carrying the link target,
    and do not emit directory members. Covered by `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | doc |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | no — shape only |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | doc |
| TestDetail12 | 12 | partially |
