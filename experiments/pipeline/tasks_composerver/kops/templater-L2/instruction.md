# Contract (L2) — templater

A Go text template named `mainTemplate` is parsed from `content` with sprig functions plus `indent`, `include`, and three channel lookups (recommended kubernetes upgrade for a version, recommended kops kubernetes version, recommended image for cloud/k8s/arch). If failOnMissing is true, missing keys are errors; otherwise missing keys may render as `<no value>` / empty per the engine. Each snippet is parsed under its filename; colliding with `mainTemplate` is an error. Render recovers panics from include/channel helpers into an error. Indent: split on `\n`, pad every non-first non-empty line with `indent` spaces, preserve newlines between lines. Include executes a named snippet with the given context map.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRenderGeneralOK` | basic interpolation |
| `TestRenderMissingValue` | failOnMissing errors |
| `TestAllowForMissingVars` | failOnMissing false allows missing |
| `TestRenderIndent` | indent skips first/empty lines |
| `TestRenderSnippet` | include named snippet |
| `TestRenderContext` | context map fields |
| `TestRenderChannelFunctions` | channel recommended version/image funcs |
| `TestRenderIntegration` | combined snippets + funcs |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/util/templater/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
