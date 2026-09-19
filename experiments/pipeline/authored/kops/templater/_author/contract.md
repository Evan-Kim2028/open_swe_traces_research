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
