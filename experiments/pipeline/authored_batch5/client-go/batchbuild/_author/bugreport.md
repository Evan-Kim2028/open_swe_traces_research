# Bug report

`internal/client`'s `batchCommandsBuilder` is stubbed: `push`, `len`,
`hasHighPriorityTask`, `buildWithLimit`, `cancel`, `reset`,
`newBatchCommandsBuilder` and the `batchCommandsEntry` helpers panic, so
no batch request can be assembled.

Expected (in-package probes, fresh builder each): pushing 2 normal
entries then `buildWithLimit(2,nil)` yields a request with RequestIds
[0 1] and an empty queue; pushing a canceled entry plus a normal one
then `buildWithLimit(10,nil)` yields 1 request with RequestIds [0] (the
canceled entry is skipped); `buildWithLimit(0,nil)` over a normal-only
queue returns nil and leaves the entry queued; `cancel(boom)` closes
each entry's `res` channel and sets its `err`; after a build and a
`reset()`, the next build's RequestIds continue the sequence ([1] after
[0]).

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
