# Bug report

Component flag strings come back empty or panic. Duration `0` is not normalized to `0s`, string-list flags lose either repeating or comma-joining, maps are unsorted, and values containing quotes are not quoted in the space-separated form.

Reproduce with:

```
go test -count=1 ./pkg/flagbuilder/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
