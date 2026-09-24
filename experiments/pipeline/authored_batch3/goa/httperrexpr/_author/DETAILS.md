# Commitments — httperrexpr

1. The error name is "HTTP error " plus the declared error name. In-tree
   coverage: none after trim. Inferable: partially.
2. JSON-RPC is decided by the response parent: a JSON-RPC API parent is
   true; a service or endpoint parent delegates to that parent's own
   JSON-RPC predicate; anything else is false. In-tree coverage: none.
   Inferable: partially.
3. On a JSON-RPC parent, reserved status codes and any mapped headers or
   cookies are each rejected with their own errors. In-tree coverage:
   trimmed tests only. Inferable: doc — a comment explains the shared
   response restriction; the reserved-code rule is documented elsewhere.
4. A code is unusable only inside -32768..-32100 exclusive of the five
   standard protocol codes (-32700, -32600, -32601, -32602, -32603);
   -32099..-32000 and everything else is allowed. In-tree coverage:
   trimmed tests only. Inferable: doc — the JSON-RPC spec defines the
   reserved block.
5. The "does not match an error defined in the ..." check names the
   scope — method, service, or API — matching the response parent kind;
   a JSON-RPC API parent reports "API". In-tree coverage: none.
   Inferable: partially.
6. Header validation for errors: an empty error type with any headers
   errors "response defines headers but error type is empty"; an object
   error type requires each header name to resolve via
   'attribute_name:header_name' and each mapped attribute to be a
   primitive or array of primitives. In-tree coverage: deleted tests
   only. Inferable: partially.
7. For a non-object error type, headers are allowed only when at most
   one is mapped ("response defines more than one headers but error type
   is not an object"); an array error type needs primitive elements; a
   map error type is always rejected. In-tree coverage: deleted tests
   only. Inferable: no.
8. Finalize resolves the error expression from the method, finalizes the
   response against it, defaults a missing body via the shared error
   body computation (then finalizes it), and maps unmapped error
   attributes to headers. In-tree coverage: none. Inferable: partially.
9. Finalize sets the response content type to the result type's
   IDENTIFIER — not its content type — only when the body is non-empty,
   the content type is unset, and the body type is a result type.
   In-tree coverage: none. Inferable: no.
10. Dup shares the resolved error expression pointer and name but
    deep-copies the response. In-tree coverage: none. Inferable:
    partially.
