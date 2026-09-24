# Bug report

Merging OpenAPI schemas panics. Expected: `Merge` fills absent fields
from the other schema, tightens min/max bounds, and appends links/required —
never overwriting present fields or concatenating composite fields. Got:
panics wherever a builder merges shared and override schemas.

Reproduce with:

```
go test -count=1 ./http/codegen/openapi/ ./http/codegen/openapi/v2/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
