# Commitments — oav3types

1. `schemafy` maps primitive kinds to OpenAPI types: 64-bit ints →
   `integer`/`int64`, 32-bit → `integer`/`int32`, `float32` →
   `number`/`float`, `float64` → `number`/`double`, bytes →
   `string`/`binary` (or an `anyOf` of base schemas when the attribute
   declares union bases), `any` → empty type. In-tree coverage:
   `TestBuildBodyTypes`, `TestMapTypes`, golden files. Inferable: doc —
   format names are spec-defined.
2. User types become `#/components/schemas/<Name>` refs: identical
   structures share one component (hash-keyed dedup), name comes from
   `Meta["openapi:typename"]`, a sole `preferredNames` entry, or
   `Meta["name:original"]`, Goified and uniquified; the component schema
   is built once and stored in `sf.schemas`. In-tree coverage:
   `TestTypesOnlyDifferByEnum`, `TestHashAttribute`,
   `TestBuildBodyTypesPreservesPrimitiveAliasComponents`. Inferable:
   partially — dedup is forced by "identical types share a name", the
   name precedence order is a choice.
3. `hashAttribute` hashes the *structure*: object member names+types
   (skipping `MustGenerate`-false members) via `orderedHash` XOR,
   array/map element recursion, user-type attribute recursion, result
   types by `Identifier`+view, validations via `hashstructure`, and a
   `seen` map guards recursion. Types differing only in validation rules
   hash differently; primitives hash by name. In-tree coverage:
   `TestHashAttribute`, `TestTypesOnlyDifferByEnum`. Inferable:
   partially — the contract "structurally identical ⇒ same hash" is
   documented, the bit-mixing is a choice.
4. Example identities thread through recursion: `at`/`member`/
   `arrayElement`/`mapValue`/`unionMember`/`field` each return a child
   schemafier whose `rand` is derived via the matching
   `ExampleGenerator` method (`field` uses `exampleFieldIdentity`).
   In-tree coverage:
   `TestHeaderSchemaAndDisplayedExampleShareIdentity`,
   `TestInitExamples*` (indirect). Inferable: partially — identity
   algebra is forced by the generator's API, which method maps to which
   node is a choice.
5. `buildBodyTypes` skips services/endpoints/types gated by
   `MustGenerate`/`mustGenerateType` (including `type:generate:force`
   service filters), assigns a default request-body description when a
   user-typed body lacks one, annotates streaming bodies, and collects
   `preferredNames` from response projections before generating. In-tree
   coverage: `TestSecuritySchemesIncludeVisibleOperationsOnly`,
   streaming tests, golden files. Inferable: partially.
6. `responseBodyProjection` detaches the body (`expr.DupAtt`) and
   projects the result type onto the pinned view (`ViewMetaKey`, default
   `default`; array bodies always project); `staticViewBody` consumes
   only the body. In-tree coverage: view/projection golden tests.
   Inferable: partially — projection is forced, detach-copy is a
   design-rule consequence.
7. `buildSSEItemSchema` emits `{data: ...}` plus optional `event`, `id`,
   `retry` properties, marks `data` required unless a selected optional
   data field, and wraps structured data as
   `contentMediaType: application/json` + `contentSchema`. In-tree
   coverage: `TestSSEItemSchemaMatchesDataContract`,
   `TestStreamingResponseStatusCodes`. Inferable: doc — OpenAPI 3.2
   sequential-media contract.
8. `ensureSchemaDescription`/`userTypeDescription` keep component
   descriptions owned by the Apikit type (attribute description only when
   the attribute shares the type's authored origin). In-tree coverage:
   `TestSharedErrorComponent*`,
   `TestUndescribedSharedErrorIgnoresLocalizedResponseDescription`.
   Inferable: partially.
9. Post-switch schema decoration resolves `Description` through
   `values.Description`, string-keyed `DefaultValue` (`toStringMap`),
   projected `Example`, `Extensions`/`AdditionalProperties` from meta,
   and the full validation block — enum, format, pattern,
   exclusive/inclusive min/max, `MinLength`→`MinItems`/`MaxLength`→
   `MaxItems` on arrays, `Required` minus `MustGenerate`-false
   attributes. In-tree coverage: `TestParamForAllowEmptyValue`, golden
   files. Inferable: partially — spec forces most mappings, exact
   filter order is a choice.
10. `toRef` spells refs `#/components/schemas/<Name>`; `toString` renders
    string/int/float64/bool map keys; `mustGenerateType` honors
    `type:generate:force` against the generated-service set; `uniquify`
    appends the smallest integer >1 that frees the name. In-tree
    coverage: golden files. Inferable: no for spellings, partially for
    the gates.

Cheat shape: real skeleton minus example-identity threading (`at`/
`member`/`field`/... return `sf`), hash = type name only (no structural
recursion, no validations), `uniquify` = identity, no preferred-name
collection effect, no view projection (`staticViewBody` returns the raw
body), no MustGenerate gates in `buildBodyTypes`, SSE schema = bare
`{data}`, no validations/extensions/`contentMediaType` wiring, no
alias/non-alias split beyond the inline case, and no name dedup in
`schemafy`. Fails `TestStreamingResponseStatusCodes` (non-baseline).
Ratio vs gold: 0.46.
