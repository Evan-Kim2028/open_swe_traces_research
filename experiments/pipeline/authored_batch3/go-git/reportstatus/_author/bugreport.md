# Bug report

The push-side status report that the server sends back after receiving a
pack is misparsed. A report telling the client its pack was rejected is
surfaced as success, per-ref rejections lose their reason text or are
swallowed wholesale when more than one ref failed, and a report that stops
before its terminator is accepted as complete. Ref lines that should be
rejected for carrying extra fields — or for missing the required failure
reason — slip through, while valid lines are refused. On the write side the
message comes out missing pieces: refs marked as failed go out looking
accepted, and the report omits its terminator.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
