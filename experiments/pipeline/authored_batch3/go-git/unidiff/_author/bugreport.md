# Bug report

Diff output is malformed. File headers print decimal instead of octal
modes, drop the index line when content and mode change together, and
emit ---/+++ lines even when only the mode changed. New and deleted files
use the wrong paths on the diff line. Hunk headers always include the
line count even for single-line ranges, the section context after @@ is
missing, and two changes separated by a small unchanged run are split
into separate hunks instead of merged. Content lines without a trailing
newline lose the end-of-file marker, and the counts in the @@ ranges are
off by one for trailing changes.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
