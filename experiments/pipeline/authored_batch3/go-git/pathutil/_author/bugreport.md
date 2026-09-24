# Bug report

The path-safety checks let disguised dangerous names through while
rejecting harmless ones. Components that hide a repository-metadata
directory name behind Unicode lookalikes or filesystem short-name forms
are accepted; plain metadata names spelled normally are refused in some
checks that should accept them. Names ending in spaces, dots, or
colon-suffixed streams bypass the reserved-name check; a device name
with an extension is treated as safe. Tilde paths expand even without a
separator, and home-directory lookups that fail return an empty path
instead of the original. Tree paths containing parent components or a
disguised metadata directory deep in the path pass validation.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
