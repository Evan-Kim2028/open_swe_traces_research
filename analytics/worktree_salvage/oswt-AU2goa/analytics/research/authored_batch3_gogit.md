# authored_batch3 — goa L0-hard units (20)

Third authoring pass over the Goa repo, same recipe as batch-2
(`authoring_hard_l0_units.md`): each unit excises one closure, deletes or
trims the in-tree tests that pin it, and ships `gold.patch` +
`cheat.patch` + full metadata (`api.md`, `DETAILS.md`, `bugreport.md`,
`closure.md`, `difficulty.md`) under
`experiments/pipeline/authored_batch3/goa/<unit>/_author/`.

Selection bias: parsing / predicate / serialisation surfaces over
orchestration — the batch-2 cohort showed those flip hardest. Cheats are
naive partial implementations: they keep signatures and obvious cases,
drop recursion through bases/references, alias unwrapping, order
insensitivity, merge/tag logic, and secondary cleanups. Every cheat
builds and stays under the 0.6 size ratio vs gold.

Overlap check vs all prior units (authored batch-1 + authored_batch2 in
the `oswt-AUgoa` worktree + batch-3 pre-existing units): **CLEAN — 0
overlaps**. Command:

```
uv run python scripts/check_unit_overlap.py --extra /home/evan/Documents/oswt-AUgoa
```

Two intra-batch file shares are symbol-disjoint and reported INFO, not
overlap: `typepred`/`compat` share `expr/types.go`; `valrules`/`atpred`
share `expr/attribute.go`. (`valrules` originally also excised
`AttributeExpr.IsSupportedValidationFormat`, which collided with
atpred's `AttributeExpr.*` marker set under the checker's granularity —
it was dropped from the closure and stays implemented.)

## Validation status

- Excised trees: `go build ./...` + `go test -run '^$'` compile checks
  pass for all 20 (build gate in `author_unit_build.py`).
- Gold restore: applying `gold.patch` to each excised tree reproduces the
  original non-test source **byte-identically** for all 20.
- Bare failure: 19/20 excised trees fail `go test` on remaining shipped
  tests (stubs panic under surviving callers). `gattr` passes the
  shipped suite — all its pinning tests were removed; the same shape
  holds for certified batch-2 units (e.g. `retrypolicy`), and the
  preflight bare-FAIL gate runs the verifier-generated hidden suite,
  not shipped tests. (`valrules` does fail shipped tests:
  `TestEvaluateAttachedJSONRPCErrorDoesNotReadPackageRoot` reaches the
  `ValidationExpr.Dup` stub through HTTP error-body finalization —
  re-measured 2026-09-20 on a freshly reconstructed excised tree.)
- Cheat: all 20 build; 14 fail remaining shipped tests, 6 pass them
  (`endparams`, `exgen`, `gattr`, `grpcresp`, `intcep`, `oatags`) — the
  hidden suite is the discriminator, as designed.
- Cheat size ratio: all 20 < 0.6 of gold's added lines.
- Solver trials: **not run** (out of scope for this batch).

## Units

| unit | closure | file(s) | syms | details | Inferable y/d/p/n | cheat | pred |
|---|---|---|---|---|---|---|---|
| atpred | attribute predicates + lookups | `expr/attribute.go` | 15 | 11 | 2/0/4/5 | 0.51 | L1 |
| bodytype | HTTP body shape derivation | `expr/http_body_types.go` | 16 | 8 | 0/0/3/5 | 0.42 | L2 |
| compat | data-type compatibility predicates | `expr/types.go` | 7 | 6 | 0/1/3/2 | 0.49 | L1 |
| endparams | endpoint params + route rules | `expr/http_endpoint.go` | 8 | 7 | 1/1/1/4 | 0.49 | L2 |
| errdiff | error-contract difference | `expr/error_contract.go` | 13 | 6 | 0/0/3/3 | 0.14 | L2 |
| exgen | example generation | `expr/example.go` | 15 | 8 | 0/0/4/4 | 0.47 | L2 |
| exid | example identity encoding | `expr/example_identity.go` | 28 | 6 | 0/0/3/3 | 0.53 | L2 |
| gattr | attribute graph copier | `expr/attribute_graph_copier.go` | 11 | 6 | 1/0/2/3 | 0.22 | L2 |
| goname | Go naming helpers | `codegen/{funcs,goify}.go` | 10 | 8 | 1/0/5/2 | 0.55 | L1 |
| gotag | Go type/tag generation | `codegen/types.go` | 6 | 6 | 1/0/1/4 | 0.52 | L2 |
| goval | Go value rendering | `codegen/go_value.go` | 17 | 10 | 0/0/5/5 | 0.60 | L2 |
| grpcresp | gRPC response expr | `expr/grpc_response.go` | 5 | 6 | 0/0/4/2 | 0.16 | L2 |
| intcep | interceptor validation | `expr/interceptor.go` | 3 | 3 | 0/0/2/1 | 0.30 | L1 |
| namedecl | codegen name declarations | `codegen/name_declaration.go` | 15 | 8 | 1/1/3/3 | 0.37 | L2 |
| oaschema | OpenAPI schema dup/merge/project | `http/codegen/openapi/{json_schema,merge}.go` | 16 | 7 | 0/0/4/3 | 0.49 | L2 |
| oatags | OpenAPI tags/extensions/marshal | `http/codegen/openapi/{tags,extensions,marshal}.go` | 8 | 5 | 0/0/2/3 | 0.54 | L1 |
| typepred | type predicates + equality | `expr/types.go` | 13 | 7 | 5/1/1/0 | 0.35 | L1 |
| unionid | union declaration/type IDs | `codegen/union.go` | 6 | 6 | 0/0/2/4 | 0.53 | L2 |
| validcode | generated validation code | `codegen/validation.go` | 12 | 8 | 1/0/4/3 | 0.51 | L2 |
| valrules | validation expr lifecycle | `expr/attribute.go` | 6 | 6 | 0/0/4/2 | 0.56 | L2 |

Inferable column = yes/doc/partially/no counts across the numbered
commitments in each `DETAILS.md`. Cheat column = added-lines ratio vs
gold (< 0.6 required). Pred = predicted flip level from `difficulty.md`.

## Per-unit notes

- **typepred** (`expr/types.go`): `As*/Is*/Equal` predicate cluster.
  Mostly Inferable: yes — predicate truth tables are recoverable from
  names + GoDoc; the hard core is UserType/ResultType unwrapping depth
  and `Equal`'s structural recursion, which the cheat flattens.
- **compat** (same file, disjoint symbols): `IsCompatible` family —
  recursion through user types and map/array element rules.
- **atpred** (`expr/attribute.go`): `AttributeExpr` predicates and
  lookup helpers (`AllRequired`, `Delete`, `FieldTag`, `Find`, …) plus
  `walkAttribute`/`unalias`/`TaggedAttribute` free functions.
- **valrules** (same file, disjoint): `ValidationExpr` lifecycle —
  bound-conflict `Validate` table, bound-tightening `Merge`, required-set
  ops, `HasRequiredOnly`, `Dup`. Cheat merges fill-empty only, skipping
  the min-tightening/max-loosening comparisons and exclusive-bound
  cross-checks.
- **bodytype**: HTTP request/response body derivation — header/param
  subtraction, JSON-RPC ID fields, result-type views, metadata
  propagation. Cheat keeps the obvious shape, drops the subtle removal
  rules.
- **endparams**: route/param extraction and validation — wildcard
  uniqueness, HEAD/WebSocket rules, trailing-slash handling all dropped
  by the cheat.
- **errdiff**: error-contract diff — order-insensitive comparison and
  effective-attribute materialization replaced by `reflect.DeepEqual`
  and shallow checks.
- **exgen**: deterministic example generation — retry loop, pattern AST
  walking, per-kind min/max arithmetic all simplified away.
- **exid**: length-prefixed identity encoding — cheat uses raw
  concatenation (ambiguous keys) and skips the seed guard; gRPC/error
  constructors left as panics.
- **gattr**: deep attribute-graph copy — cheat does shallow copies
  without graph reconnection or deep value copies.
- **goname**: `Goify`/`SnakeCase`/`WrapText`/comment helpers — cheat is
  a naive title-caser (delegating to `internal/codegenname` would have
  been a correct implementation, so it was written in place).
- **gotag**: Go type names + struct tags — cheat drops tag merging and
  naming conventions.
- **goval**: Go value rendering — cheat renders literals but skips
  pointer-local declarations, sorted map keys, custom-type validation,
  union constructor resolution. Largest cheat in the batch (0.60, at
  threshold) because the literal table itself is sizable.
- **grpcresp**: `GRPCResponseExpr` lifecycle — cheat keeps EvalName/Dup,
  drops Finalize's metadata/status merging.
- **intcep**: smallest closure (3 funcs) — interceptor arg validation;
  cheat checks only obvious shapes.
- **namedecl**: `NameDeclaration` accessors + validators — cheat keeps
  small accessors, leaves validators stubbed.
- **oaschema**: `Schema.Dup`/`Merge` + example projection — cheat does
  shallow field copies (no reflection-walked deep copy) and fill-empty
  merge without bound tightening.
- **oatags**: tag meta parsing + extension merge + marshal — cheat
  parses tags without version gating/ExternalDocs detail and merges
  extensions without JSON value decoding.
- **unionid**: union ID rendering — cheat concatenates names flat, no
  length-prefixing or cycle detection.
- **validcode**: validation code emission — cheat drops nil guards,
  alias handling, exclusive bounds, and several format constants.

## Known limitations

- `expr`/`codegen` test suites have pre-existing baseline failures on
  the unmodified source (2 codegen tests) — unrelated to these units.
- `oaschema` required a hand-written `merge.go` test stub via `test_add`
  because the generic `duplicatePointer[T any]` defeats the stubber's
  regex; it stays implemented in the unit.
- `typepred` and `exid` delete `expr/project_test.go` (package-init
  vars call stubbed funcs); its helpers were verified file-local.
- `goval`'s cheat sits exactly at the 0.6 ratio ceiling — accepted by
  the validator but worth a glance during verification.
- Inferable annotations are author judgment; the DETAILS gate will
  re-grade them downstream.
