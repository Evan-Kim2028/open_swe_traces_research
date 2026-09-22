# Contract (L2) — httpresp

An HTTP response expression maps a method result onto a status code, header
and cookie mappings, and a body. Its generic name in error messages is an
HTTP-flavoured label that also carries the parent expression's name when a
parent is set.

Preparation installs empty mapped attributes for both headers and cookies when
either is unset.

Validation rules, in order of what they guard:

- A status of exactly 0 is reported as an undefined response status. A body is
  forbidden only for informational statuses (100–199), no-content and
  not-modified; every other status — including reset-content and the other
  redirect codes — allows one. That check is skipped entirely for streaming
  endpoints and fires only when the computed endpoint body is non-empty.
- On a JSON-RPC endpoint a success response may not map result attributes to
  headers or cookies; non-empty mappings on either produce dedicated errors.
- A text content type (the plain-text and HTML content types) forces the
  result — or an explicit body — to be a string or bytes type, unless the
  endpoint skips body encode/decode.
- Header names must resolve against the result type — for a result type the
  lookup is per-view, and a missing name reports the `attribute_name:header_name`
  correspondence notation qualified as all views. Mapped header values must be
  primitives or arrays of primitives; mapped cookie values must be primitives
  only — an array mapped to a cookie is rejected. An unmatched cookie name is
  reported with the same correspondence notation.
- On a non-object result, header or cookie mappings are allowed only while at
  most one is mapped; an array result mapped to a header must have primitive
  elements, and an array result mapped to a cookie is always an error.
- An explicit body under the skip-body-encode/decode flag is an error; with no
  explicit body the computed body must be empty under the same flag. A body
  attribute named through the origin-attribute metadata, or every field of an
  object body, must exist in the result type, reported by name.

Finalization records the endpoint as parent, wraps a non-empty object body in
a generated user type named after the service and endpoint with a fixed
ResponseBody suffix, records the wrapped type's original name under its own
metadata key, splits mapped field names at their header separator, and adds
required body fields to the body's validation. The response content type falls
back to the result type's own content type only when the response did not set
one, and headers and cookies inherit their attribute definitions from the
result attribute. Duplicating a response copies the scalar fields — status
code and whether it was set, description, content type, parent, metadata —
and deep-copies the body, headers and cookies.

Error responses get one more pass: for built-in error-result bodies only,
every attribute not mapped to the body or headers is mapped to a header under
a fixed `apikit-attribute-<name>` key — attributes already mapped keep their
declared header name, requiredness is propagated — and a non-object error
result maps under a single `apikit-attribute` key when headers and body are
both empty.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | generic name is an HTTP label that carries the parent's name when set |
| `TestDetail02` | prepare installs empty mapped attributes for headers and cookies |
| `TestDetail03` | body forbidden only for 1xx, no-content, not-modified; streaming skips; fires only on a non-empty computed body |
| `TestDetail04` | status 0 is reported as an undefined response status |
| `TestDetail05` | JSON-RPC success response may not map result attributes to headers or cookies |
| `TestDetail06` | text content types force the result or explicit body to string or bytes |
| `TestDetail07` | header names resolve against the result type (per-view for result types); header values primitive or primitive arrays, cookie values primitive only |
| `TestDetail08` | non-object result: at most one header or cookie; array-to-header needs primitive elements; array-to-cookie always errors |
| `TestDetail09` | explicit body under skip flag errors; computed body must be empty under it; body attributes must exist in the result type |
| `TestDetail10` | finalize wraps a non-empty object body in a generated user type with the ResponseBody suffix, records the original name, propagates requiredness |
| `TestDetail11` | content type falls back to the result type's own content type only when unset |
| `TestDetail12` | dup copies scalar fields and deep-copies body, headers and cookies |
| `TestDetail13` | error-result bodies only: unmapped attributes map to `apikit-attribute-<name>` headers, mapped names skipped, requiredness propagated |
