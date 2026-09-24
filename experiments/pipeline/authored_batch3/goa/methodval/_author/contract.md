# Contract (L2) — methodval

A method expression validates and finalizes one service method: its payload,
streaming payload, result, streaming result, security requirements, errors and
interceptors.

Validation merges diagnostics in a fixed order — payload, streaming payload,
result, then streaming result only when it is a different object than the
result — followed by the requirements, the errors, and the interceptors.

Security is checked against the effective requirements: the method's own, else
the service's, else the API's. Each scheme kind demands its own credential
field in the payload, found through the credential tags the DSL's security
helpers write: basic auth needs a username field AND a password field; an API
key needs the field tagged with the key prefix namespaced by that scheme's
name; bearer, JWT and OAuth2 each need their own tagged field. The inverse
also holds: a payload field carrying a credential tag with no matching scheme
kind in the requirements is an error naming the kind — one check per kind,
with the API-key check matching by prefix. Every scope listed in a requirement
must exist in at least one of that requirement's credential-kind schemes; a
missing scope is named in the error.

A payload field marked as a security attribute — any of the five fixed
credential tags, or any tag carrying the API-key prefix — must be a string or
a named string type (named types count after unwrapping to their underlying
attribute), must not carry a field-type override, and must not declare a
default value, because missing credentials must remain missing. Each violation
names the field. Credential tags are discovered by walking the payload's own
metadata, then recursively through its base user types, then its type — so a
named-type payload carrying credential fields satisfies a requirement.

Interceptors declared on the method, the service and the API are merged into
the method's client and server lists in precedence order — method first, then
service, then API — deduplicated by name so the tighter scope wins.

Finalization defaults a missing payload, streaming payload or result to an
empty attribute, finalizes the results including their result types, inherits
service-declared errors whose names the method does not already declare,
finalizes each error — authored user-type errors plainly, the rest typed to
the method — and inherits requirements from the service, else the API, with
each inherited requirement duplicated rather than shared; an explicit
no-security declaration instead produces a single requirement holding one
kindless scheme.

The streaming predicates follow the method's stream kind: the payload streams
on client-streaming and bidirectional kinds, the result streams on
server-streaming and bidirectional kinds, and a method has mixed results only
when both result objects are present and are different objects.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | validation merges payload, streaming payload, result, streaming result (when a different object), then requirements, errors, interceptors |
| `TestDetail02` | a security-attribute payload field must be String or named String, no field-type override, no default — each violation names the field |
| `TestDetail03` | effective requirements are the method's own, else the service's, else the API's |
| `TestDetail04` | each scheme kind demands its credential field: basic needs username and password, API key the scheme-namespaced tag, bearer/JWT/OAuth2 their own |
| `TestDetail05` | every scope in a requirement must exist in one of its credential-kind schemes; missing scopes named |
| `TestDetail06` | a credential-tagged field with no matching scheme kind errors naming the kind; the API-key check matches by prefix |
| `TestDetail07` | security attribute means the five fixed credential tags plus any API-key-prefixed tag |
| `TestDetail08` | a named type counts as String only after unwrapping user types to their underlying attribute |
| `TestDetail09` | method/service/API interceptors merge into the method lists in precedence order, deduplicated by name |
| `TestDetail10` | credential tags are found through the payload's own meta, its base user types, then its type |
| `TestDetail11` | finalize defaults nil payload/streaming-payload/result to empty attributes, inherits service errors by name, inherits requirements, no-security gives one kindless scheme |
| `TestDetail12` | inherited requirements are duplicated, not shared |
| `TestDetail13` | payload streams on client and bidirectional kinds, result on server and bidirectional; mixed results needs both result objects present and different |
