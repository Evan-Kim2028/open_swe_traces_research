# Bug report

`git archive` output is broken in several ways: extracted files have
loose permissions (777 everything, or empty modes), `git
get-tar-commit-id` finds no commit in archives that should carry one,
archiving `v1.0:subdir` produces the whole tree instead of the
subdirectory, symlinks are written as regular files containing their
target path, and a path filter that matches nothing still produces an
archive instead of failing. Remote callers can also pass `--prefix=../x`
and get traversal entries written.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
