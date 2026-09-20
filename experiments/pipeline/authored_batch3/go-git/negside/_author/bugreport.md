# Bug report

Two side-messages of the wire protocol are corrupt. The shallow-boundary
updates sent during fetch come back with hashes in the wrong lists,
malformed lines are silently skipped instead of refused, lines with junk
where the object id should be are accepted, and the stream is considered
complete without its end marker. The push options sent alongside a push
are emitted with spurious line breaks inside each payload, partial output
reaches the wire before a bad option is noticed, and whitespace-only
options are refused even though plain spaces are legal.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
