# Bug report

Rename detection in tree diffs is wrong in several ways. Pure renames —
same content, new path — show up as a delete plus an add instead of a
modify. Near-copies (a file moved then edited) are never detected at all.
When several files were deleted and one added with identical content, the
pair chosen is arbitrary rather than the closest name. And on large
changesets the detector can be observed to either run forever or produce
nonsense pairings, instead of declining gracefully.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
