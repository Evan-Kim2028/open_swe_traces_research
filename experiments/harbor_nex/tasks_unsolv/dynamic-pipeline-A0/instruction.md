# Missing behavior

The pipelined write buffer has a mutable in-memory map, an in-flight flush
buffer, and a remote store. A lookup must return the newest value in that
order: mutable buffer, then the buffer currently being flushed, then the
remote batch getter. A missing key is “not exist”.

Writes go into the mutable buffer. A flush swaps that buffer into the
in-flight slot and continues in the background: further writes must not
block on the flush finishing unless the mutable buffer has grown past the
force-flush size. A flush is skipped when there are too few keys or too
little data (minimum 10000 keys and 16MiB, or 128MiB to force). After a
flush, a waiter must observe the background error. Concurrent writers
must not race; `go test -race` must stay clean.

Lookups after thousands of inserts must stay fast under parallel readers.
A single global mutex around a linear scan of every stored pair is too
slow: 5000 sequential inserts then parallel lookups must stay under the
ns/op ceiling recorded from the reference implementation (gold time × 3).

A lookup that always reports not-exist is wrong: expected the stored
value, actual not exist.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
go test -count=1 -timeout 15m -race ./internal/unionstore/
go test -count=1 -timeout 15m -bench=^BenchmarkPipelinedGet$ -benchtime=5000x ./internal/unionstore/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
