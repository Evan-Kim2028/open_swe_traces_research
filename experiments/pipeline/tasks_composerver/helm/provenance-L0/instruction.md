# Bug report

`helm install --verify` / `helm package --sign` fail: provenance verification cannot run — signing produces nothing, verification errors or panics, and digest/keyring handling is broken, so signed charts cannot be validated.

Reproduce with:

```
go test -count=1 ./pkg/provenance/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
