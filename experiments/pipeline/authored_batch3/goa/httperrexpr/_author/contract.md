# Contract (L2) — httperrexpr

An HTTP error expression maps one declared method/service/API error onto a
response: status code, header and cookie mappings, and a body. Its generic
name in error messages is an HTTP-flavoured label carrying the declared error
name.

Whether the mapping lives on JSON-RPC is decided by the response's parent: a
JSON-RPC API parent means yes; a service or endpoint parent delegates to that
parent's own JSON-RPC predicate; anything else means no. On a JSON-RPC parent
two extra rules apply, each producing its own error: mapped headers and mapped
cookies are forbidden outright, and the status code may not come from the
reserved block — codes −32768 through −32100 are unusable except the five
standard protocol codes, while −32099 through −32000 and everything outside
the block are allowed.

Every mapping must resolve: the named error must exist in the scope matching
where the response was declared — the method, the service, or the API (a
JSON-RPC API-level mapping reports the API scope) — and the error names the
missing definition.

Header mappings are checked against the error type. Any headers on an empty
error type are rejected. On an object error type each header name must resolve
to an attribute of the error type — the `attribute_name:header_name` notation
identifies the correspondence — and each mapped attribute must be a primitive
or an array of primitives. On a non-object error type at most one header may
be mapped; an array error type additionally needs primitive elements, and a
map error type is always rejected.

Finalization resolves the error expression from the endpoint's method,
finalizes the response against it, computes a default body — the error
attributes not already mapped to headers — when none was defined, and maps any
still-unmapped error attributes to headers. When the body is non-empty, the
response content type unset, and the body type a result type, the response
content type falls back to that result type's identifier. Duplicating the
mapping shares the resolved error expression pointer and the name but
deep-copies the response.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | the generic name is an HTTP-flavoured label carrying the declared error name |
| `TestDetail02` | JSON-RPC is decided by the response parent: JSON-RPC API true, service/endpoint delegate, anything else false |
| `TestDetail03` | on a JSON-RPC parent, reserved status codes and any mapped headers or cookies are each rejected |
| `TestDetail04` | reserved means −32768..−32100 exclusive of the five standard protocol codes; −32099..−32000 and all else allowed |
| `TestDetail05` | an unmatched error name reports the scope — method, service, or API — matching the response parent kind |
| `TestDetail06` | empty error type plus headers errors; object error type requires each header to resolve and be primitive or primitive array |
| `TestDetail07` | non-object error type allows at most one header; array needs primitive elements; map is always rejected |
| `TestDetail08` | finalize resolves the method's error expression, finalizes the response, defaults the body to unmapped attributes, maps the rest to headers |
| `TestDetail09` | response content type falls back to the body result type's identifier only when body non-empty, content type unset, body a result type |
| `TestDetail10` | duplication shares the resolved error expression and name but deep-copies the response |
