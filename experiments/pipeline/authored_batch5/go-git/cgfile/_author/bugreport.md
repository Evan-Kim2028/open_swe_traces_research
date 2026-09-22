# Bug report

Reading commit-graph files accepts corrupt inputs and mis-handles valid
ones. Files with truncated chunk tables, duplicated chunks, or chunk counts
that disagree with the fanout are opened without complaint and only blow up
later. Octopus merges read their extra parent list without bounds, and
corrected committer timestamps past the overflow marker are never resolved.
In a chained setup, files that exist but fail to parse trigger a fallback as
if they were missing, parents of merged chains stop answering lookups they
should delegate, and closing a chain leaks the parent file.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
