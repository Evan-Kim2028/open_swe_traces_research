# Bug report

The root gate for the privileged test suites is broken. Suites that should
be skipped without `-test.root` instead run (and crash on missing
privileges), or the gate never lets a legitimately-root run proceed, or it
exits with the wrong status — green when the run lacked privileges, or
killing the binary on the skip path with a failure code.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
