# Bug report

A raw single-key lookup of user key `key` in keyspace 0x1092 went out as
the bare user key. expected the four-byte prefix `0x72 0x00 0x10 0x92`
followed by the user key, actual the bare bytes. expected keyspace id
0x10203 from a well-formed header, actual 0xffffffff.

Locating a region whose range keys are not well-formed also retried with
backoff. expected a hard failure and zero backoff, actual backoff
continued (or the lookup succeeded).

Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./internal/apicodec/ ./internal/locate/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
