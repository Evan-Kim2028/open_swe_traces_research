# Bug report

Rendering with value overrides produces wrong values: user `-f`/`--set` input does not override chart defaults (or overrides too aggressively), subchart values/globals are missing, and null deletion semantics are broken, so `helm template` output is wrong.

Reproduce with:

```
go test -count=1 ./pkg/chart/common/util/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
