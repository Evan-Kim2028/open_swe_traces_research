# Commitments — oav2spec

1. Marshaling a spec object emits its own fields AND the entries of its
   `Extensions` map as top-level keys (`x-*` vendor extensions live at
   the same level as `info`, `paths`, etc.). In-tree coverage:
   `TestExtensions` golden-compares documents carrying endpoint
   extensions. Inferable: doc — the Swagger/OpenAPI extension convention
   requires sibling keys, not a nested object.
2. Marshaling never dispatches back into the same `MarshalJSON`/
   `MarshalYAML` method: bodies convert to the shadow alias type
   (`_Info`, `_Path`, `_Operation`, `_Parameter`, `_Response`,
   `_SecurityDefinition`) before delegating. In-tree coverage: every
   marshaling test would hang/crash on recursion. Inferable: partially —
   the need to break recursion is forced, the alias-type mechanism is a
   choice.
3. An explicitly empty `SecurityRequirements` (`SecurityRequirements{}`)
   serializes as `security: []`; a nil one is omitted entirely
   (`json:"security,omitzero"` consults `IsZero`, which reports `s ==
   nil`, not `len(s) == 0`). In-tree coverage:
   `TestOperationSecurityMarshal` asserts both cases in JSON and YAML.
   Inferable: yes — `dsl.NoSecurity()` must produce an explicit empty
   array per the spec, distinct from "no security field".
4. YAML marshaling returns a `map[string]any`-compatible value with
   extension keys merged at the top level, produced by round-tripping
   through `yaml.Marshal`/`yaml.Unmarshal` when extensions exist and
   returning the alias value directly otherwise. In-tree coverage:
   `TestNoSecurityOverrides*` yaml subtests, `TestExtensions` yaml
   files. Inferable: partially — the merge contract is forced, the
   round-trip mechanism is a choice.
5. `IsZero` is defined on the `SecurityRequirements` slice type, not on
   `Operation`; nil-vs-empty distinction is the only signal it reads.
   In-tree coverage: `TestOperationSecurityMarshal`. Inferable: yes.

Cheat shape: naive `json.Marshal(_X(x))` bodies without the extension
merge for four types, `MarshalJSON` deleted for `Info`/`Path`, all six
`MarshalYAML` methods deleted (yaml falls back to struct tags:
extensions dropped, `security: []` lost to `omitempty`), `IsZero`
answered by `len(s) == 0`. Fails `TestExtensions`,
`TestNoSecurityOverridesAPISecurity`,
`TestNoSecurityOverridesServiceSecurity`, `TestOperationSecurityMarshal`.
Ratio vs gold: 0.54.
