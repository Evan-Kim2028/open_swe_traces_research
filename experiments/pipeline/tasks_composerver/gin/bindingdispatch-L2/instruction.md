# Contract (L2) — bindingdispatch

A lookup picks the binder for a request: method GET always returns the form binder regardless of content type; otherwise the content type selects among JSON, XML (two registered MIME strings), ProtoBuf, MsgPack (two MIME strings), YAML (two MIME strings), TOML, multipart form, and BSON; anything else — including the urlencoded form type and empty content type — falls back to the plain form binder. The internal validate hook calls the configured Validator and returns nil when no validator is installed.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestBindingDefault` | every MIME row of the dispatch table returns its binder; GET overrides content type |
| `TestValidationDisabled` | with no validator installed the validate hook is a no-op |



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

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
