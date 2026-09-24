# contracts_goa3 — L2 contracts for the 11 verified goa batch-3 units

2026-09-21. Units: `experiments/pipeline/authored_batch3/goa/<unit>/`, copied from
`oswt-VFbatch3goa`. Each arrived with `gold.patch`, `bugreport.md`, a verified hidden
suite (`tests/hidden/expr/<unit>_bb_test.go`, one `TestDetailNN` per DETAILS.md line) and
no `_author/contract.md` — so the stager could not mount them at any rung.

Two other goa dirs in the batch are excluded: `apiexpr` and `resultview` have no `tests/`
tree (never verified), so no contract was written for them.

## Shape

Contracts follow `reconciled_contracts.md`: prose invariants first, then a coverage table
with exactly one row per `TestDetailNN`, each row derived from (hidden assertion, gold
hunk). B7 held throughout — no file names, line numbers or private identifiers; the only
backticked literals are public DSL meta keys (`rpc:tag`, `struct:field:type`,
`grpc:stream:compat`, `attribute_name:header_name`, `apikit-attribute-<name>`), the
exported method under test, and wire-visible values (`show`, `/`, `{…}`).

## Coverage: commitments vs hidden assertions

| unit | DETAILS commitments | coverage rows | hidden tests | unjustifiable assertions |
|---|---:|---:|---:|---|
| attachsvc | 9 | 9 | 9 | none |
| defval | 12 | 12 | 12 | none |
| grpcend | 14 | 14 | 14 | none |
| httperrexpr | 10 | 10 | 10 | none |
| httpresp | 13 | 13 | 13 | none |
| httpsvc | 12 | 12 | 12 | none |
| methodval | 13 | 13 | 13 | none |
| rootval | 14 | 14 | 14 | none (see fix below) |
| secschemes | 12 | 12 | 12 | none |
| svcerrors | 8 | 8 | 8 | none |
| svcexpr | 12 | 12 | 12 | none |
| **total** | **129** | **129** | **129** | **0** |

Row counts are exact both ways: every hidden test names a row, every row names a test
(verified by cross-reading each `TestDetailNN` body against its row, not just by count).

## Derivability spot-checks (literals the tests assert)

The no-arbitrary-literal rule bans *committing* an unguessable literal; where a test
asserts one anyway, derivability was confirmed against the solver-visible tree:

| literal | asserted by | solver-visible source |
|---|---|---|
| `grpc:stream:compat` = `"v1"` | grpcend TestDetail03/05 | `streamCompatLegacy` const + doc comment survive excision in `expr/grpc_endpoint.go` |
| `name:original` meta key | httpresp TestDetail10 | producers in `expr/user_type.go`, `expr/http_body_types.go`; consumers in `http/codegen/openapi`, `grpc/codegen` |
| `struct:pkg:path` | rootval TestDetail12 | documented in `dsl/meta.go` |
| `struct:error:name` | svcerrors TestDetail06 | written by the DSL `ErrorName` helper |
| `apikit-attribute-<name>` | httpresp TestDetail13 | stated in the contract itself (row 13) |
| `API` default root name | rootval TestDetail02 | **was not derivable** — lived only inside excised `root.go`; contract amended to name it |

## Fixes applied during reconciliation

- `rootval`: prose said "a plain default name" while `TestDetail02` asserts the exact
  name `API`. Contract now states `the default name `API`` (DETAILS line 2 commits it;
  the literal exists nowhere in the solver-visible tree). No other gaps found.

## Flagged units

None. Every hidden assertion maps to a derivable commitment.
