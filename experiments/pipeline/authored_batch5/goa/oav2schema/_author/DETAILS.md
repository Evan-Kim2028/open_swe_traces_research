# Commitments — oav2schema

1. `BuildAttributeSchema` attaches the builder's collected `definitions`
   to the returned schema as `Defs` when non-empty. In-tree coverage:
   `TestBuildAttributeSchemaKeepsDefinitionsSeparate`. Inferable: yes —
   the doc comment says the schema "includes every named definition".
2. Each build keeps its own definitions and assigned names — two builds
   of the same design do not share state, and result types reused across
   views keep the name first assigned. In-tree coverage:
   `TestBuildsAreSafeToRunTogether`, `TestBuildsKeepDefinitionsSeparate`.
   Inferable: yes — file doc states it.
3. Definition names come from `TypeName`, overridden by
   `Meta["openapi:typename"]` (Goified), and prefixed with the Goified
   prefix when the unprefixed name is already taken; user types are
   stored under `typeName`, result types under the projected name via a
   non-mutating `renamedResultType` copy. In-tree coverage: definition
   golden tests, `TestBuildsKeepDefinitionsSeparate`. Inferable:
   partially — the meta key is a documented override, the prefix/dedup
   order is a choice.
4. Primitive kinds map to OpenAPI types: signed/unsigned 64-bit →
   `integer`/`int64`, 32-bit → `integer`/`int32`, `float32` →
   `number`/`float`, `float64` → `number`/`double`, bytes →
   `string`/`byte`, `any` → empty type. In-tree coverage: golden schema
   output, `TestNamedPrimitiveParamsAndHeadersUseOpenAPIBaseTypes`.
   Inferable: doc — OpenAPI format names are spec-defined.
5. Map schemas emit a typed `additionalProperties` schema only when the
   (alias-unwrapped) key type is string and the element type is not
   `any`; otherwise `additionalProperties: true`. In-tree coverage:
   `TestBuildAttributeSchemaKeepsValuesForNamedStringMapKeys`.
   Inferable: doc — JSON Schema semantics.
6. Union schemas emit one `anyOf` member per value, each an object whose
   properties are the type key (string enum of the member name) and the
   value key (the member schema, with the member's own validations), both
   listed in `required`. In-tree coverage:
   `TestAttributeTypeSchemaCorrelatesUnionDiscriminatorAndValue`.
   Inferable: partially — discriminator-object shape is forced by the
   union contract, key names come from the union expression.
7. `initSchemaValidation` copies enum, format, pattern, min/max
   (inclusive and exclusive), and remaps `MinLength`/`MaxLength` to
   `MinItems`/`MaxItems` when the attribute type is an array; `Required`
   names skip attributes whose meta says `MustGenerate` is false.
   In-tree coverage: `TestValidations` golden output. Inferable:
   partially — the length→items remap is spec-forced, the meta filter
   is a documented generation gate.
8. Object properties skip attributes failing `MustGenerate`; schemas
   carry `Title` (`Mediatype identifier: <id>` for results, `typeName`
   for user types), `Media.Type` = result identifier, `DefaultValue`,
   resolved `Description`, projected `Example`, `Extensions` from meta,
   and `AdditionalProperties` from meta. In-tree coverage: golden file
   tests, `TestSharedErrorDefinition*`,
   `TestDefaultFacadeMatchesWithValues`. Inferable: partially —
   presence is spec/doc-driven, exact title text is a choice.

Cheat shape: real builder skeleton minus `Defs` attachment, name
overrides/dedup, `Title`/`Media`, `DefaultValue`/`Description`/
`Example`/`Extensions`, alias unwrap on map keys, union member
`required`/validations, exclusive min/max, `float`/`double` formats,
`any` empty-type, `MinItems`/`MaxItems` remap, and `MustGenerate`
filtering. Fails `TestSharedErrorDefinitionDescription` (non-baseline).
Ratio vs gold: 0.58.
