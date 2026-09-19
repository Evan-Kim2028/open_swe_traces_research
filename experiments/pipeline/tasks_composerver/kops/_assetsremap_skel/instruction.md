# Bug report

Container image and file remapping is gone: docker-hub vs registry-host proxy rules, the “apply registry rewrite twice and it must converge” behavior, comma escaping in file URLs, and hash lookup/caching all panic. Concurrent remap of many images races or drops entries, and sorted asset lists come back empty.

Reproduce with:

```
go test -count=1 ./pkg/assets/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
