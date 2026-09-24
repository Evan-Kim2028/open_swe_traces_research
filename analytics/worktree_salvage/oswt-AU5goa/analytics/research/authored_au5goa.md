# authored_au5goa — goa (20 units)

Authoring run `AU5goa` (worktree `oswt-AU5goa`, branch `au5goa`).
Units live at `experiments/pipeline/authored_au5goa/goa/<unit>/_author/`.
Every unit verified locally with `/tmp/au5goa/verify.py`: excised tree builds
and its surviving suite shows zero non-baseline failures, `gold.patch` applies
to the immutable excised snapshot and restores green, `cheat.patch` fails at
least one non-baseline test. Per-package and per-test baselines
(`/tmp/au5goa/baseline{,_enum}/`) subtract pre-existing failures (rebranding
artifacts, `protoc-gen-go` missing from PATH, openapi golden-file drift) —
a panic aborts a Go test binary mid-package, so whole-package baselines alone
undercount; per-test enumeration was required.

## Units in authoring order

| # | unit | closure (file) | surface type | rejections before | cheat fails (non-baseline) | cheat ratio |
|---|------|----------------|--------------|-------------------|----------------------------|-------------|
| 1 | goval | `codegen/go_value.go` — Go literal/default rendering | serialization | 0 | 5 tests (map key order, pointer locals, custom defaults) | 0.53 |
| 2 | unionid | `codegen/union.go` — structural union type IDs | serialization | 1 | 4 tests (recursive/user-type/pointer shape in IDs) | 0.52 |
| 3 | protoids | `codegen/protobuf.go` — protobuf identifier mangling | serialization | 1 | `TestProtobufFieldNames`, `TestProtobufNames` | 0.36 |
| 4 | attcopy | `expr/attribute_graph_copier.go` — detached attribute copies | transformation | 7 | 3 tests (detach, view rewire, originals map) | 0.38 |
| 5 | errcontract | `expr/error_contract.go` — error-contract equivalence | predicate | 7 | 6 tests (bases/references materialization, order-insensitivity) | 0.12 |
| 6 | oav2spec | `http/codegen/openapi/v2/openapi.go` — v2 marshal methods | serialization | 8 | `TestExtensions`, 3 security-override tests | 0.54 |
| 7 | oav3spec | `http/codegen/openapi/v3/openapi.go` — v3 marshal methods | serialization | 8 | 3 security-override tests (`security: []` emission) | 0.53 |
| 8 | oav2schema | `http/codegen/openapi/v2/json_schema.go` — v2 schema builder | serialization | 8 | `TestSharedErrorDefinitionDescription` | 0.58 |
| 9 | oav3types | `http/codegen/openapi/v3/types.go` — v3 schemafier | serialization | 8 | `TestStreamingResponseStatusCodes` | 0.46 |
| 10 | oavers | `http/codegen/openapi/versions.go` — version/path meta selection | parsing + predicate | 15 | `TestSpecs` | 0.14 |
| 11 | compat | `http/codegen/compatibility.go` — released entry-point package guard | predicate | 15 | `TestReleasedHTTPFileFunctionsUsePlannedPackage` | 0.47 |
| 12 | intercept | `expr/interceptor.go` — interceptor access validation | predicate | 15 | `TestInterceptorExpr_Validate` | 0.37 |
| 13 | oafiles | `http/codegen/openapi/files.go` — spec file emission + YAML date quoting | serialization | 15 | `TestToYAMLQuotesDateShapedStrings` | 0.58 |
| 14 | evalxpr | `eval/expression.go` — DSLFunc/TopExpr/ToExpressionSet | predicate + conversion | 15 | `TestInvalidArgError`, `TestToExpressionSet`, `TestTooFewArgError` | 0.44 |
| 15 | oamarshal | `http/codegen/openapi/marshal.go` — extension-merged JSON/YAML | serialization | 15 | `TestExtensions`, `TestMarshalJSONPreservesLargeIntegers`, `TestValidations` | 0.36 |
| 16 | oamerge | `http/codegen/openapi/merge.go` — two-level schema merge | transformation | 15 | `TestSchemaMergeAndDupPreserveAnyOf`, `TestValidations` | 0.54 |
| 17 | oaerr | `http/codegen/openapi/error_example.go` — error content-type + example | serialization | 15 | `TestStreamingResponseStatusCodes` | 0.34 |
| 18 | oatags | `http/codegen/openapi/tags.go` — tag meta parsing + marshal | parsing + serialization | 15 | `TestExtensions`, `TestTagsFromExpr` | 0.55 |
| 19 | oaext | `http/codegen/openapi/extensions.go` — `x-*` extension extraction | parsing | 20 | `TestExtensions`, `TestExtensionsFromMethod` | 0.45 |
| 20 | oavalues | `http/codegen/openapi/values.go` — copy-on-write Values store | transformation | 20 | `TestNewV2WithValues`, `TestNewWithValues`, `TestValues` | 0.34 |

Surface mix: 9 serialization, 4 parsing (incl. mixed), 4 predicate-ish
(validation/guard/conversion), 3 transformation. Parses/predicates/
serializers dominate per the funnel measurement.

## Inferable breakdown

122 commitments total: **yes 16, doc 39, partially 57, no 10**.
`no` entries are confined to exact spellings — error-message text
(`intercept`, `evalxpr`) and literal formats (`goval`, `protoids`,
`oav3types`); the verifier asserts shape only. `partially` dominates: most
semantics are documented at behavior level while the precise mechanism
(merge tables, key choices, regex shapes) is a design detail.

## Rejected candidates (20) and reasons

| candidate | file | rejection |
|-----------|------|-----------|
| goname | `codegen/codegenname.go` | caller fan-out: Goify reached by 63 files |
| imports | `codegen/imports.go` | 327 tests reach UserTypeLocation/GetMetaType transitively |
| genimports | `codegen/import_spec.go` (etc.) | 236 panicking tests — import planning is backbone |
| walkattr | `expr/walk.go` | ~60 `http/codegen` tests reach `Walk` transitively |
| expr/types | `expr/types.go` | all `Is*`/`As*` predicates — every eval path |
| valeff | `expr/validation_effective.go` | 216 panics — EffectiveValidation on eval path |
| exident | `expr/example_identity.go` | 468 panics — example identity is everywhere |
| oav3refs | `http/codegen/openapi/v3/ref.go` | unverifiable: `$ref` serialization shape pinned only by baseline-broken golden tests; non-baseline tests compare self-consistent renders or extract unrelated fields |
| namedecl | `codegen/name_*.go` | 95 panics |
| usertype | `codegen/user_type.go` | 133 panics |
| httptypes | `http/codegen/types.go` | ~100+ panics |
| typedef | `codegen/typedef.go` | ~100+ panics |
| httpfuncs | `http/codegen/funcs.go` | ~100+ panics |
| httppaths | `http/codegen/paths.go` | ~100+ panics |
| symbols | `codegen/symbol_table.go` | ~100+ panics |
| secscheme | `security/scheme.go` | no test files in `security/` — surface unpinned |
| oaproj | `http/codegen/openapi/response_projection.go` | 0 trims: no non-baseline test reaches the stub |
| validcode | `codegen/validation.go` | 233 panics |
| ssecode | `http/codegen/sse.go` | 102 panics |
| wirecat | `http/codegen/wire_catalog.go` | 147 panics |

Additionally `valplan`, `jrplan`, `bodytypes` were driven green
(excised+gold verified) but not used — once the `http/codegen/openapi` vein
filled the quota, their cheat-authoring cost (200–400-line partial
implementations) was not worth a marginal unit.

## Overlap check

`scripts/check_unit_overlap.py` does not exist (job template referenced it
aspirationally); the de-facto check is a claimed-file map extracted from every
`*/_author/closure.md` across `authored/`, `authored_batch2/` (sibling
worktree) and `authored_batch3/`: **66 goa files claimed**. All 20 new units
excise files outside that set and pairwise-disjoint among themselves — zero
collisions. `oav3refs` was the only candidate that *looked* overlapped-adjacent
(shares the v3 package with `oav3types`/`oav3spec`) but it was rejected on
verifiability grounds, not overlap.

## Search-cost trajectory

Rejects-before-acceptance climbed in steps, not smoothly: 0 → 1 → 7 → 8 →
15 → 20. The expensive middle (units 4–9) absorbed the `expr/`/`codegen/`
backbone cluster — every central helper fans out to hundreds of tests. The
`http/codegen/openapi/` shared package then yielded 11 of the last 13 units
with almost no dead ends (each candidate was pre-filtered by caller count and
a dedicated test file). Cost per accepted unit was **flat-to-falling** at the
end (units 10–20: 11 accepts, 5 new rejections, plus 3 green spares) — the
run stopped because the 20-unit quota filled, not because returns diminished.

## Cheat design notes

- Thin-delegation closures (`oav2spec`, `oav3spec`, `oafiles`, `oaext`,
  `oaerr`, `oavers`, `compat`): the cheat deletes or degrades methods — a
  solver "skipping the hard part" (extension merge, YAML path, validation
  checks). Deletion cheats beat reimplementation for ratio on small golds.
- Big-builder closures (`oav2schema`, `oav3types`, `oamerge`, `oavalues`):
  cheats keep a running skeleton and drop the subtle commitments (identity
  threading, dedup, field-guards, version gates) — ratio lands 0.34–0.58.
- `oamerge` needed a second cheat pass: dropping *scalar* merge rows didn't
  move any test; dropping composite rows (`AnyOf`/`Items`/`Enum`) failed the
  dedicated merge test. Lesson: the cheat must remove what a *test asserts*,
  not what looks removable.
- `evalxpr`: the direct unit test swallows panics via `recover`; the cheat is
  pinned only through indirect DSL-execution tests — noted in closure.md.
