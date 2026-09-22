# Closure — exid

`expr/example_identity.go` — deterministic example-seed identity:
kind-tagged, length-framed keys for every example surface plus
structural descent.

Symbols stubbed: all `*ExampleIdentity` constructors, `Seed`, `Member`,
`ArrayElement`, `MapKey`, `MapValue`, `UnionMember`, and the framing
helpers `newExampleIdentity`, `methodExampleIdentity`,
`exampleIdentityInt`, `appendExampleIdentitySegment`,
`ExampleIdentity.append`.

Tests removed: `expr/project_test.go` deleted (package vars call the
stubbed constructors at init).
