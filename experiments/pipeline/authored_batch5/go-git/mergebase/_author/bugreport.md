# Bug report

Merge-base queries return wrong or incomplete answers. When one commit is
a direct ancestor of the other, the query still walks the whole graph and
sometimes returns extra, non-best bases. On diverged histories with
criss-cross merges, bases that are themselves reachable from other bases
are reported alongside the real answer. Asking for the independent set of
a commit list returns duplicates or members that another member's history
covers. Ancestor checks against an identical commit disagree with the
reference tool.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
