# Bug report

`helm dependency update`/`build` fails: it cannot resolve declared dependencies to a lock file — repo lookups, local file:// paths, OCI references, and aliased charts all error or produce an empty/wrong lock, and the lock digest is unstable.

Reproduce with:

```
go test -count=1 ./internal/resolver/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
