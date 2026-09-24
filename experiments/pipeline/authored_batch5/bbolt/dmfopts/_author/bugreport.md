# Bug report

The dmflakey failure-injection helper is broken at its configuration edge.
Feature options come back nil or clobber each other's settings, unsupported
filesystems sail through validation, a constructed device reports the wrong
device-mapper path or forgets which filesystem it was built with, and image
creation tramples existing files or reaches for external tools before
checking its inputs.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
