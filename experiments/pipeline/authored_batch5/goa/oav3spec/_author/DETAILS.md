# Commitments — oav3spec

1. Marshaling a spec object emits its own fields AND the entries of its
   `Extensions` map as top-level keys (`x-*` vendor extensions sit beside
   `info`, `paths`, `operationId`, ...). In-tree coverage: golden
   document tests (`TestFiles`, `TestFilesV32` — baseline-broken),
   extension-bearing fixture tests. Inferable: doc — the OpenAPI
   extension convention requires sibling keys.
2. Marshaling never dispatches back into the same method: bodies convert
   to shadow alias types (`_Info`, `_PathItem`, `_Operation`,
   `_Parameter`, `_Response`, `_SecurityScheme`) before delegating to
   `openapi.MarshalJSON`/`openapi.MarshalYAML`. In-tree coverage: every
   render would crash on recursion. Inferable: partially — recursion
   must be broken, the alias mechanism is a choice.
3. An explicitly empty `SecurityRequirements` serializes as
   `security: []`; a nil one is omitted (`json:"security,omitzero"`
   consults `IsZero`, which reports `s == nil`, not `len(s) == 0`). In
   YAML the same distinction is preserved by the marshaled value. In-tree
   coverage: `TestOperationSecurityMarshal` (json+yaml),
   `TestNoSecurityOverrides*` (assert `security` present-but-empty on
   `NoSecurity` operations). Inferable: yes — `dsl.NoSecurity()` must
   produce an explicit empty array per the spec.
4. `setExample` assigns `Example`; `setExamples` assigns `Examples` —
   the two fields are mutually exclusive per the spec and are driven by
   `initExamples` via the `exampler` interface. In-tree coverage:
   `TestInitExamplesUsesReplacementDescription`, parameter/media-type
   example tests. Inferable: yes — the interface contract names them.
5. YAML marshaling returns a top-level-merged map value (round-tripped
   through `yaml.Marshal`/`yaml.Unmarshal` when extensions exist, the
   alias value otherwise). In-tree coverage: yaml subtests of the
   security and example tests. Inferable: partially.

Cheat shape: all six `MarshalJSON` and all seven `MarshalYAML` methods
deleted outright (default struct marshaling drops `Extensions` entirely
and `yaml:"security,omitempty"` drops the explicit empty array),
`IsZero` answered by `len(s) == 0` (drops `security: []` from JSON),
`setExample`/`setExamples` implemented verbatim. Fails
`TestNoSecurityOverridesAPISecurity`,
`TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`.
Ratio vs gold: 0.53.
