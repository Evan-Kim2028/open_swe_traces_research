# Bug report

Addon channel YAML no longer loads: empty files error, raw manifests are not wrapped, and names are wrong. Version filtering and “which of two same-named addons wins” (id / content hash / generation, never replace a newer generation) are broken, and the “do we need an update / install PKI” decision is always wrong.

Reproduce with:

```
go test -count=1 ./channels/pkg/channels/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
