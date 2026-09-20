# Bug report

Reflog history files parse and write incorrectly. Lines that end the file
without a newline are dropped, blank lines abort the whole read, and a
read that hits a malformed line throws away the entries already decoded.
Signatures whose names contain stray angle brackets are misparsed, and a
message containing newlines or tabs is written verbatim so it breaks the
file format. Entries with no message get a stray trailing tab, timezone
offsets outside the usual range are rejected even though the digits are
fine, and batch decodes crash on a nil source instead of reporting it.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
