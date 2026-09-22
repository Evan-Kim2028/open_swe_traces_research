# Bug report

`util` request-source composition panics: `GetRequestSource`,
`BuildRequestSource`, the setters, context helpers and `IsInternalRequest`
are stubbed, so RPCs cannot be labeled with their request source.

Expected: `BuildRequestSource(true, "test", "lightning")` =
`"internal_test_lightning"`, `(false, "test", "lightning")` =
`"external_test_lightning"`, `(false, "test", "")` = `"external_test"`,
`(false, "", "lightning")` = `"external_unknown_lightning"`, `(true, "",
"")` = `"unknown"`. A nil or all-empty `RequestSource` yields `"unknown"`.
`IsInternalRequest("internal_gc")` is true, `IsInternalRequest("external_x")`
false, `IsRequestSourceInternal(nil)` false.
`RequestSourceFromCtx(WithInternalSourceType(ctx, "gc"))` =
`"internal_gc"`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
