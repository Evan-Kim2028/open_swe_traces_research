# Contract (L2) — svcerrors

A service expression validates and finalizes its methods and the errors
declared on them.

Its diagnostic name is an unnamed-service placeholder when empty, else the
quoted service name; its hash is the service name behind a fixed prefix.

Service validation merges each declared error's own validation first, then
checks inline method errors for consistency. The consistency check covers
only errors whose constructor is generated — an error whose type is not an
authored user type, or is the built-in error-result type; errors backed by an
authored user type do not participate because the shared type fixes the
contract. Service-level errors seed the seen-set first; then each method's
generated-constructor error must match the previously seen same-named
contract — when the mismatch is in qualifier settings the error names which
settings differ, any other difference produces the generic same-contract
message naming the error and service.

An error's own validation enforces the error-name marker: at most one field
per possibly-nested type may carry it, and the marked field must be a string
and required — each violation names the offending field or type.

Error finalization keeps the type a user type so a value can be generated: an
authored non-error-result user type receives the error-name marker on its
attribute unless one of its object fields already carries it; an error whose
type is not a user type has its attribute wrapped in a generated user type
named after the error. Method-declared errors instead get a generated user
type keyed by the method-plus-error example identity, reusing the origin of
the first previously-generated same-named error found walking the service's
methods strictly before the current one — so same-named inline errors in
later methods share the earlier one's origin, while differently-named errors
never do.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | diagnostic name is an unnamed placeholder or the quoted name; the hash is the name behind a fixed prefix |
| `TestDetail02` | service validation merges each error's own validation before the inline-consistency check |
| `TestDetail03` | generated-constructor means not an authored user type, or the built-in error result type; authored-type errors do not participate |
| `TestDetail04` | service errors seed the seen-set; each generated-constructor method error must match the seen contract — qualifier differences name the settings, others the generic contract message |
| `TestDetail05` | at most one error-name-marked field per type; it must be a required string |
| `TestDetail06` | finalize marks an authored non-error-result user type unless a field already carries the marker; a non-user-type error is wrapped in a user type named after it |
| `TestDetail07` | method-typed error finalization builds a generated user type reusing the first earlier same-named error's origin in method order |
| `TestDetail08` | the origin search walks methods strictly before the current one and only picks same-named generated user types |
