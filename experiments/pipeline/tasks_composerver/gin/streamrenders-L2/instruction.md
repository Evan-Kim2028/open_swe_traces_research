# Contract (L2) — streamrenders

Reader streams: it sets its own content type, sets Content-Length only when ContentLength >= 0, then writes each custom header only where the response has none set, then copies the body through. Data writes Content-Length only when the payload is non-empty and then writes the bytes. Redirect validates the status code — anything outside the 3xx range is a panic, except 201 Created which is also allowed — then delegates to the standard redirect helper with the request and location. String writes plain text: with data args it renders the format verb, without args it writes the format string literally (a lone '%' stays literal). All of them leave a pre-set Content-Type alone.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRenderReader/TestRenderReaderNoContentLength` | stream copy with Content-Length only when >= 0 |
| `TestRenderData/TestRenderDataContentLength/TestRenderDataError` | bytes written with Content-Length only when non-empty |
| `TestRenderRedirect` | status-code validation panic outside 3xx/201 and normal redirect otherwise |
| `TestRenderString/TestRenderStringLenZero` | format-with-args vs literal format string |
| `TestRenderWriteError` | write failures propagate |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./render/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
