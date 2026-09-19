# Contract (L2) — bindingdispatch

A lookup picks the binder for a request: method GET always returns the form binder regardless of content type; otherwise the content type selects among JSON, XML (two registered MIME strings), ProtoBuf, MsgPack (two MIME strings), YAML (two MIME strings), TOML, multipart form, and BSON; anything else — including the urlencoded form type and empty content type — falls back to the plain form binder. The internal validate hook calls the configured Validator and returns nil when no validator is installed.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestBindingDefault` | every MIME row of the dispatch table returns its binder; GET overrides content type |
| `TestValidationDisabled` | with no validator installed the validate hook is a no-op |
