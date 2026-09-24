# Bug report

Patches built from tree or commit changes are broken in several ways.
Diffs touching submodules either drop the change entirely or try to read
the gitlink as a file and fail. Binary files produce garbage text hunks
instead of being marked binary. File stats miscount: renamed files lose
the "old => new" naming, trailing lines without a newline are dropped
from the counts, and the graph column overflows on large changes instead
of scaling down. Cancelling mid-build is ignored — the walk runs to
completion.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
