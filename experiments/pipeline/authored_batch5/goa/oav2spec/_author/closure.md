# Closure — oav2spec

`http/codegen/openapi/v2/openapi.go` — Swagger 2.0 document serialization:
per-type `MarshalJSON`/`MarshalYAML` methods that delegate to
`openapi.MarshalJSON`/`openapi.MarshalYAML` through anti-recursion alias
types and merge per-object `Extensions` into the top-level document map,
plus `SecurityRequirements.IsZero` driving `json:",omitzero"`.

Symbols stubbed: `Info.MarshalJSON`, `Path.MarshalJSON`,
`Operation.MarshalJSON`, `Parameter.MarshalJSON`, `Response.MarshalJSON`,
`SecurityDefinition.MarshalJSON`, `Info.MarshalYAML`, `Path.MarshalYAML`,
`Operation.MarshalYAML`, `Parameter.MarshalYAML`, `Response.MarshalYAML`,
`SecurityDefinition.MarshalYAML`, `SecurityRequirements.IsZero`.

Exported entry points: the marshalers fire implicitly when the generated
swagger template executes (`openapiv2.Files` → template render →
`json.Marshal`/`yaml.Marshal` of the spec tree) and when callers marshal
sub-objects directly.

Tests removed: none wholesale. Trimmed (all panic on the stubs while
rendering or marshaling spec objects): `TestBuildsKeepDefinitionsSeparate`
(build_isolation_test.go); `TestNoSecurityOverridesAPISecurity`,
`TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`
(builder_test.go); `TestExtensions`, `TestSections`, `TestValidations`
(files_test.go); `TestDefaultFacadeMatchesWithValues`
(public_api_test.go); `TestAuthoredExampleFixturesIgnoreRandomizer`,
`TestFiles`, `TestFilesV32` (v3 files_test.go — render v3 specs whose
responses embed this package's serialization via shared helpers).

Behavioral commitments: extension keys merge into the top-level serialized
object; `security: []` is emitted for an explicitly empty requirement and
omitted for nil; marshaling never recurses through the method itself;
output shape matches the Swagger 2.0 schema.

Blast radius: 11 trimmed tests across 5 files in 2 packages. Size: 13
one-line bodies.
