# Bug report

Tree-diff results misbehave. A change with both endpoints populated is
reported as an insertion, a deletion sorts under the destination path
instead of the source path, and a change whose entry points at a
directory crashes instead of yielding no files. Reading a changed file
keeps carriage returns at the end of every line, drops the final line
when it lacks a newline, and mislabels short text files containing a NUL
byte as binary-free. Malformed changes panic when asked for their action
instead of reporting it.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
