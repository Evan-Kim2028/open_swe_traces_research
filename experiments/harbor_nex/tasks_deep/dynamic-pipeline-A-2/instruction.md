# Bug report

I wrote a key and then looked it up. The lookup said the key does not
exist. expected the value I just stored, actual not exist.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
go test -count=1 -timeout 15m -race ./internal/unionstore/
go test -count=1 -timeout 15m -bench=^BenchmarkPipelinedGet$ -benchtime=5000x ./internal/unionstore/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
