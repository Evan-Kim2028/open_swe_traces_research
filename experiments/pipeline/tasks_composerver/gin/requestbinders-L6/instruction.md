# Contract (L2) — requestbinders

Four request sources each feed the shared field-mapping engine, then validate. The urlencoded+multipart binder parses the full form (query plus body, multipart up to a 32MB memory cap) and tolerates a not-multipart request; the post-form binder uses only body form values; the query binder uses only URL query values; the multipart binder presents the request itself as the value source so file fields bind from the uploaded file list while ordinary fields bind from form values. Header binding uses the `header` tag and looks keys up by their canonical MIME header form (case-insensitive in, canonical form used for lookup). URI binding takes a params map and uses the `uri` tag. Every binder reports a short lowercase Name(). Validation runs after mapping in all of them.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestBindingForm/TestBindingForm2` | urlencoded+multipart binder maps parsed form incl. query values |
| `TestBindingFormPost/TestFormPostBindingFail` | post-form binder reads body values only |
| `TestBindingQuery/TestBindingQuery2/Fail` | query binder maps URL values and propagates conversion errors |
| `TestHeaderBinding` | header binder maps canonicalized header keys via the header tag |
| `TestUriBinding/TestUriInnerBinding` | uri binder maps the params map via the uri tag, nested structs included |
| `TestBindingFormFilesMultipart/TestBindingFormMultipart` | multipart binder binds files and values from the same request |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./binding/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestBBFormBinderQueryAndBody`, `TestBBFormBinderToleratesNonMultipart`, `TestBBFormPostBodyOnly`, `TestBBFormPostIgnoresQuery`, `TestBBQueryBinderURLOnly`, `TestBBQueryConversionError`, `TestBBHeaderBinderCanonical`, `TestBBHeaderBinderCaseInsensitive`, `TestBBUriBinderParams`, `TestBBUriNestedStruct`, `TestBBBinderNamesLowercase`, `TestBBValidationAfterMapping`, `TestBBValidationAfterMappingRandom`: TestBBFormBinderQueryAndBody

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
